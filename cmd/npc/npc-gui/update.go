package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"ehang.io/nps/lib/version"
	"github.com/egoist/mygo"
)

const (
	updateRepo     = "yisier/nps"
	updateAPIURL   = "https://api.github.com/repos/" + updateRepo + "/releases/latest"
	updateDownBase = "https://github.com/" + updateRepo + "/releases/download"
)

// UpdateInfo 版本检查结果，供前端展示
type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseNotes    string `json:"releaseNotes"`
	PublishedAt     string `json:"publishedAt"`
	DownloadURL     string `json:"downloadUrl"`
	AssetName       string `json:"assetName"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	PublishedAt string    `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

// guiAssetName 当前平台对应的 GUI 发布包文件名
func guiAssetName() string {
	return fmt.Sprintf("npc-gui-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
}

// compareVersion 比较两个版本号（忽略前导 v）。a<b 返回 -1，a>b 返回 1，相等返回 0
func compareVersion(a, b string) int {
	ai, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".", ""))
	bi, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".", ""))
	switch {
	case ai < bi:
		return -1
	case ai > bi:
		return 1
	}
	return 0
}

// fetchLatestRelease 拉取 GitHub 最新 release 信息
func fetchLatestRelease() (*ghRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, updateAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "npc-gui-updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取版本信息失败: HTTP %d", resp.StatusCode)
	}
	rl := new(ghRelease)
	if err := json.NewDecoder(resp.Body).Decode(rl); err != nil {
		return nil, err
	}
	if rl.TagName == "" {
		return nil, errors.New("无法解析最新版本号")
	}
	return rl, nil
}

func findAsset(rl *ghRelease, name string) (*ghAsset, error) {
	for i := range rl.Assets {
		if rl.Assets[i].Name == name {
			return &rl.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("发布资源中未找到 %s，当前平台暂不支持自动更新", name)
}

// CheckForUpdate 检查是否有新版本可用（检测升级）
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	rl, err := fetchLatestRelease()
	if err != nil {
		return nil, err
	}
	latest := strings.TrimPrefix(rl.TagName, "v")
	current := version.VERSION

	assetName := guiAssetName()
	downloadURL := ""
	if asset, err := findAsset(rl, assetName); err == nil {
		downloadURL = asset.BrowserDownloadURL
	}

	return &UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: compareVersion(current, latest) < 0,
		ReleaseNotes:    rl.Body,
		PublishedAt:     rl.PublishedAt,
		DownloadURL:     downloadURL,
		AssetName:       assetName,
	}, nil
}

// DownloadAndInstallUpdate 下载最新版本并热替换当前可执行文件
func (a *App) DownloadAndInstallUpdate() error {
	rl, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	latest := strings.TrimPrefix(rl.TagName, "v")
	if compareVersion(version.VERSION, latest) >= 0 {
		return fmt.Errorf("当前已是最新版本 %s，无需更新", version.VERSION)
	}

	assetName := guiAssetName()
	asset, err := findAsset(rl, assetName)
	if err != nil {
		return err
	}

	slog.Info("开始下载更新包", "url", asset.BrowserDownloadURL)

	exePath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}

	tempDir := filepath.Join(os.TempDir(), "npc-gui-update")
	_ = os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return err
	}

	zipPath := filepath.Join(tempDir, assetName)
	if err := downloadFile(asset.BrowserDownloadURL, zipPath); err != nil {
		return err
	}

	if err := replaceFromZip(zipPath, exePath, tempDir); err != nil {
		return err
	}

	slog.Info("更新完成，请重启程序生效")
	return nil
}

// RestartApp 重启当前程序（更新完成后调用）
func (a *App) RestartApp() error {
	mygo.App.Relaunch()
	return nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "npc-gui-updater")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载更新包失败: HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// replaceFromZip 从 zip 更新包中取出可执行文件并热替换正在运行的程序
func replaceFromZip(zipPath, exePath, tempDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开更新包失败: %w", err)
	}
	defer zr.Close()

	exeBase := filepath.Base(exePath)
	var found *zip.File
	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if strings.EqualFold(name, exeBase) || strings.EqualFold(name, "npc-gui") || strings.EqualFold(name, "npc-gui.exe") {
			found = f
			break
		}
	}
	if found == nil {
		return fmt.Errorf("更新包中未找到可执行文件 %s", exeBase)
	}

	rc, err := found.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	newBin := filepath.Join(tempDir, "npc-gui"+guiExeExt())
	out, err := os.Create(newBin)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	out.Close()
	chmodExecutable(newBin, 0o755)

	return replaceExecutableFile(newBin, exePath)
}

func guiExeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// replaceExecutableFile 用新文件替换正在运行的可执行文件。
// Windows 下运行中的 exe 不能直接覆盖，先改名备份再替换。
func replaceExecutableFile(srcBin, destBin string) error {
	if srcFi, err := os.Stat(srcBin); err == nil {
		if dstFi, err := os.Stat(destBin); err == nil {
			if os.SameFile(srcFi, dstFi) {
				return nil
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(destBin), 0o755); err != nil {
		return err
	}

	// 将当前程序改名为 .old 备份（Windows 上运行中的文件允许改名）
	if _, err := os.Stat(destBin); err == nil {
		bak := destBin + ".old"
		_ = os.Remove(bak)
		if err := os.Rename(destBin, bak); err != nil {
			return fmt.Errorf("无法备份当前程序 %s: %w", destBin, err)
		}
	}

	// 同文件系统 rename 原子替换；跨卷回退为拷贝
	if err := os.Rename(srcBin, destBin); err != nil {
		if _, copyErr := copyFileForUpdate(srcBin, destBin); copyErr != nil {
			// 尽力恢复旧程序
			bak := destBin + ".old"
			if _, statErr := os.Stat(bak); statErr == nil {
				_ = os.Rename(bak, destBin)
			}
			return fmt.Errorf("替换可执行文件失败: %w", copyErr)
		}
		_ = os.Remove(srcBin)
	}
	return nil
}

func copyFileForUpdate(src, dest string) (int64, error) {
	srcF, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcF.Close()
	dstF, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer dstF.Close()
	return io.Copy(dstF, srcF)
}

func chmodExecutable(path string, mode os.FileMode) {
	if runtime.GOOS != "windows" {
		_ = os.Chmod(path, mode)
	}
}
