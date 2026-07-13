package install

import (
	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/version"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/c4milo/unpackit"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Keep it in sync with the template from service_sysv_linux.go file
// Use "ps | grep -v grep | grep $(get_pid)" because "ps PID" may not work on OpenWrt
const SysvScript = `#!/bin/sh
# For RedHat and cousins:
# chkconfig: - 99 01
# description: {{.Description}}
# processname: {{.Path}}
### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.DisplayName}}
# Description:       {{.Description}}
### END INIT INFO
cmd="{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}"
name=$(basename $(readlink -f $0))
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"
[ -e /etc/sysconfig/$name ] && . /etc/sysconfig/$name
get_pid() {
    cat "$pid_file"
}
is_running() {
    [ -f "$pid_file" ] && ps | grep -v grep | grep $(get_pid) > /dev/null 2>&1
}
case "$1" in
    start)
        if is_running; then
            echo "Already started"
        else
            echo "Starting $name"
            {{if .WorkingDirectory}}cd '{{.WorkingDirectory}}'{{end}}
            $cmd >> "$stdout_log" 2>> "$stderr_log" &
            echo $! > "$pid_file"
            if ! is_running; then
                echo "Unable to start, see $stdout_log and $stderr_log"
                exit 1
            fi
        fi
    ;;
    stop)
        if is_running; then
            echo -n "Stopping $name.."
            kill $(get_pid)
            for i in $(seq 1 10)
            do
                if ! is_running; then
                    break
                fi
                echo -n "."
                sleep 1
            done
            echo
            if is_running; then
                echo "Not stopped; may still be shutting down or shutdown may have failed"
                exit 1
            else
                echo "Stopped"
                if [ -f "$pid_file" ]; then
                    rm "$pid_file"
                fi
            fi
        else
            echo "Not running"
        fi
    ;;
    restart)
        $0 stop
        if is_running; then
            echo "Unable to stop, will not attempt to start"
            exit 1
        fi
        $0 start
    ;;
    status)
        if is_running; then
            echo "Running"
        else
            echo "Stopped"
            exit 1
        fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
exit 0
`

const SystemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}
[Service]
LimitNOFILE=65536
StartLimitInterval=5
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
{{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
{{if and .LogOutput .HasOutputFileSupport -}}
StandardOutput=file:/var/log/{{.Name}}.out
StandardError=file:/var/log/{{.Name}}.err
{{- end}}
Restart=always
RestartSec=120
[Install]
WantedBy=multi-user.target
`

func UpdateNps() {
	destPath, err := downloadLatest("server")
	if err != nil {
		log.Println("下载更新失败：", err)
		return
	}
	//复制文件到对应目录
	if _, err := copyStaticFile(destPath, "nps"); err != nil {
		log.Println("替换服务端文件失败：", err)
		return
	}
	fmt.Println("Update completed, please restart")
}

func UpdateNpsNew() {
	latest, err := fetchLatestVersion()
	if err != nil {
		log.Println("获取最新版本失败：", err)
		return
	}
	fmt.Println("最新版本为：", latest)
	if compareVersion(version.VERSION, latest) >= 0 {
		fmt.Println("当前已是最新版本，无需更新")
		return
	}
	tempDir := filepath.Join(common.GetAppPath(), "temp")
	destPath, err := downloadLatest2("server", tempDir)
	if err != nil {
		log.Println("下载更新失败：", err)
		return
	}
	//复制文件到对应目录
	if err := copyStaticFileReplaceNps(destPath, common.GetAppPath()); err != nil {
		log.Println("替换服务端文件失败：", err)
		return
	}
	fmt.Println("更新成功，请重启服务")
}

func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/yisier/nps/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	rl := new(release)
	if err := json.Unmarshal(b, rl); err != nil {
		return "", err
	}
	if rl.TagName == "" {
		return "", errors.New("无法解析最新版本号")
	}
	return rl.TagName, nil
}

func compareVersion(a, b string) int {
	ai, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimPrefix(a, "v"), ".", ""))
	bi, _ := strconv.Atoi(strings.ReplaceAll(strings.TrimPrefix(b, "v"), ".", ""))
	if ai < bi {
		return -1
	}
	if ai > bi {
		return 1
	}
	return 0
}

func UpdateNpc() {
	destPath, err := downloadLatest("client")
	if err != nil {
		log.Println("下载更新失败：", err)
		return
	}
	//复制文件到对应目录
	if _, err := copyStaticFile(destPath, "npc"); err != nil {
		log.Println("替换客户端文件失败：", err)
		return
	}
	fmt.Println("Update completed, please restart")
}

func UpdateNpcNew() {
	latest, err := fetchLatestVersion()
	if err != nil {
		log.Println("获取最新版本失败：", err)
		return
	}
	fmt.Println("最新版本为：", latest)
	if compareVersion(version.VERSION, latest) >= 0 {
		fmt.Println("当前已是最新版本，无需更新")
		return
	}
	tempDir := filepath.Join(common.GetAppPath(), "temp")
	destPath, err := downloadLatest2("client", tempDir)
	if err != nil {
		log.Println("下载更新失败：", err)
		return
	}
	if err := copyStaticFileReplaceNpc(destPath, common.GetAppPath()); err != nil {
		log.Println("替换客户端文件失败：", err)
		return
	}
	fmt.Println("更新成功，请重启客户端")
}

type release struct {
	TagName string `json:"tag_name"`
}

func downloadLatest(bin string) (string, error) {
	return downloadAndUnpack(bin, "")
}

func downloadLatest2(bin string, path string) (string, error) {
	return downloadAndUnpack(bin, path)
}

// downloadAndUnpack fetches the latest release package for the current OS/arch.
// Releases ship as .tar.gz (see build.assets.sh / release.yml).
func downloadAndUnpack(bin, unpackPath string) (string, error) {
	data, err := http.Get("https://api.github.com/repos/yisier/nps/releases/latest")
	if err != nil {
		return "", err
	}
	defer data.Body.Close()
	if data.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取版本信息失败: HTTP %d", data.StatusCode)
	}
	b, err := ioutil.ReadAll(data.Body)
	if err != nil {
		return "", err
	}
	rl := new(release)
	if err := json.Unmarshal(b, rl); err != nil {
		return "", err
	}
	if rl.TagName == "" {
		return "", errors.New("无法解析最新版本号")
	}
	ver := rl.TagName
	fmt.Println("the latest version is", ver)
	filename := runtime.GOOS + "_" + runtime.GOARCH + "_" + bin + ".tar.gz"
	downloadUrl := fmt.Sprintf("https://github.com/yisier/nps/releases/download/%s/%s", ver, filename)
	fmt.Println("download package from ", downloadUrl)
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: HTTP %d %s", resp.StatusCode, downloadUrl)
	}
	destPath, err := unpackit.Unpack(resp.Body, unpackPath)
	if err != nil {
		return "", err
	}
	if bin == "server" {
		destPath = strings.Replace(destPath, "/web", "", -1)
		destPath = strings.Replace(destPath, `\web`, "", -1)
		destPath = strings.Replace(destPath, "/views", "", -1)
		destPath = strings.Replace(destPath, `\views`, "", -1)
	} else {
		destPath = strings.Replace(destPath, `\conf`, "", -1)
		destPath = strings.Replace(destPath, "/conf", "", -1)
	}
	return destPath, nil
}

func copyStaticFile(srcPath, bin string) (string, error) {
	// nps web UI is embedded in the binary; no web/ files to copy.
	binPath, _ := filepath.Abs(os.Args[0])
	srcBin := filepath.Join(srcPath, bin)
	if common.IsWindows() {
		srcBin += ".exe"
	}
	if _, err := os.Stat(srcBin); err != nil {
		return "", fmt.Errorf("更新包中未找到可执行文件 %s: %w", srcBin, err)
	}
	if !common.IsWindows() {
		if _, err := copyFile(srcBin, "/usr/bin/"+bin); err != nil {
			if _, err := copyFile(srcBin, "/usr/local/bin/"+bin); err != nil {
				return "", err
			}
			binPath = "/usr/local/bin/" + bin
		} else {
			binPath = "/usr/bin/" + bin
		}
	} else {
		destBin := filepath.Join(common.GetAppPath(), bin+".exe")
		if err := replaceExecutable(srcBin, destBin); err != nil {
			return "", err
		}
		binPath = destBin
	}
	chMod(binPath, 0755)
	return binPath, nil
}

func copyStaticFileReplaceNps(srcPath, descPath string) error {
	// Web UI is embedded in the binary; only replace the executable.
	return replaceBinFromPackage(srcPath, descPath, "nps")
}

func copyStaticFileReplaceNpc(srcPath, descPath string) error {
	return replaceBinFromPackage(srcPath, descPath, "npc")
}

func replaceBinFromPackage(srcPath, descPath, bin string) error {
	srcBin := filepath.Join(srcPath, bin)
	destBin := filepath.Join(descPath, bin)
	if common.IsWindows() {
		srcBin += ".exe"
		destBin += ".exe"
	}
	// Prefer replacing the actually running binary when its basename matches.
	if exe, err := os.Executable(); err == nil {
		if filepath.Base(exe) == filepath.Base(destBin) {
			destBin = exe
		}
	}
	if _, err := os.Stat(srcBin); err != nil {
		// unpackit may return a nested root dir; search one level if needed
		if found, findErr := findBinInDir(srcPath, filepath.Base(srcBin)); findErr == nil {
			srcBin = found
		} else {
			return fmt.Errorf("更新包中未找到可执行文件 %s: %w", srcBin, err)
		}
	}
	if err := replaceExecutable(srcBin, destBin); err != nil {
		return err
	}
	chMod(destBin, 0755)
	// Clean temp package; keep parent temp dir if still in use
	_ = os.RemoveAll(srcPath)
	return nil
}

func findBinInDir(root, name string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}
		if info.Name() == name {
			found = path
			return errors.New("found")
		}
		return nil
	})
	if found != "" {
		return found, nil
	}
	if err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}

// replaceExecutable places srcBin at destBin. On Windows a running executable
// cannot be overwritten, but it can be renamed aside first.
func replaceExecutable(srcBin, destBin string) error {
	if _, err := os.Stat(srcBin); err != nil {
		return fmt.Errorf("源文件不存在: %s: %w", srcBin, err)
	}
	if err := os.MkdirAll(filepath.Dir(destBin), 0755); err != nil {
		return err
	}

	// Move the current binary out of the way when present (required on Windows
	// while the process is still running).
	if _, err := os.Stat(destBin); err == nil {
		bak := destBin + ".old"
		_ = os.Remove(bak)
		if err := os.Rename(destBin, bak); err != nil {
			return fmt.Errorf("无法备份当前程序 %s: %w", destBin, err)
		}
	}

	// Same filesystem: rename is atomic. Fall back to copy across volumes.
	if err := os.Rename(srcBin, destBin); err != nil {
		if _, copyErr := copyFile(srcBin, destBin); copyErr != nil {
			// Best-effort restore of previous binary
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

func InstallNpc() {
	path := common.GetInstallPath()
	if !common.FileExists(path) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
	if _, err := copyStaticFile(common.GetAppPath(), "npc"); err != nil {
		log.Fatalln(err)
	}
}

func InstallNps() string {
	path := common.GetInstallPath()
	log.Println("install path:" + path)
	if !common.FileExists(path) {
		MkidrDirAll(path, "conf")
		// not copy config if the config file is exist
		if err := CopyDir(filepath.Join(common.GetAppPath(), "conf"), filepath.Join(path, "conf")); err != nil {
			log.Fatalln(err)
		}
		chMod(filepath.Join(path, "conf"), 0766)
	}
	binPath, err := copyStaticFile(common.GetAppPath(), "nps")
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("install ok!")
	log.Println("Web UI is embedded in the nps binary; no web/ directory is required")
	log.Println("The new configuration file is located in", path, "you can edit them")
	if !common.IsWindows() {
		log.Println(`You can start with:
nps start|stop|restart|uninstall|update
anywhere!`)
	} else {
		log.Println(`You can copy executable files to any directory and start working with:
nps.exe start|stop|restart|uninstall|update
now!`)
	}
	chMod(common.GetLogPath(), 0777)
	return binPath
}

func InstallNpsToCurrentDir() string {
	path := common.GetAppPath()
	log.Println("install path:" + path)
	log.Println("install ok!")
	chMod(filepath.Join(path, "nps.log"), 0777)

	if !common.IsWindows() {
		path = filepath.Join(path, "nps")
	} else {
		path = filepath.Join(path, "nps.exe")
	}
	return path
}

func MkidrDirAll(path string, v ...string) {
	for _, item := range v {
		if err := os.MkdirAll(filepath.Join(path, item), 0755); err != nil {
			log.Fatalf("Failed to create directory %s error:%s", path, err.Error())
		}
	}
}

func CopyDir(srcPath string, destPath string) error {
	//检测目录正确性
	if srcInfo, err := os.Stat(srcPath); err != nil {
		fmt.Println(err.Error())
		return err
	} else {
		if !srcInfo.IsDir() {
			e := errors.New("SrcPath is not the right directory!")
			return e
		}
	}
	if destInfo, err := os.Stat(destPath); err != nil {
		return err
	} else {
		if !destInfo.IsDir() {
			e := errors.New("DestInfo is not the right directory!")
			return e
		}
	}
	err := filepath.Walk(srcPath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if !f.IsDir() {
			destNewPath := strings.Replace(path, srcPath, destPath, -1)
			log.Println("copy file ::" + path + " to " + destNewPath)
			copyFile(path, destNewPath)
			if !common.IsWindows() {
				chMod(destNewPath, 0766)
			}
		}
		return nil
	})
	return err
}

// 生成目录并拷贝文件
func copyFile(src, dest string) (w int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()
	//分割path目录
	destSplitPathDirs := strings.Split(dest, string(filepath.Separator))

	//检测时候存在目录
	destSplitPath := ""
	for index, dir := range destSplitPathDirs {
		if index < len(destSplitPathDirs)-1 {
			destSplitPath = destSplitPath + dir + string(filepath.Separator)
			b, _ := pathExists(destSplitPath)
			if b == false {
				log.Println("mkdir:" + destSplitPath)
				//创建目录
				err := os.Mkdir(destSplitPath, os.ModePerm)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}
	}
	dstFile, err := os.Create(dest)
	if err != nil {
		return
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}

// 检测文件夹路径时候存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func chMod(name string, mode os.FileMode) {
	if !common.IsWindows() {
		os.Chmod(name, mode)
	}
}
