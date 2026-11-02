package install

import (
	"net/http"
	"sync"
	"time"

	"ehang.io/nps/lib/version"
)

const (
	// 版本检查成功后的缓存时间，避免 Web 页面刷新频繁请求 GitHub API
	versionCheckCacheTTL = 30 * time.Minute
	// 版本检查失败后的缓存时间，避免 GitHub 不可达时反复等待超时
	versionCheckErrorTTL = 5 * time.Minute
)

// metadataClient 用于获取版本信息，避免请求长时间阻塞
var metadataClient = &http.Client{Timeout: 10 * time.Second}

var (
	versionCheckMu     sync.Mutex
	versionCheckLatest string
	versionCheckErr    error
	versionCheckAt     time.Time
)

// CheckLatestVersion 查询最新版本并判断是否有更新。
// 结果会缓存一段时间，避免 Web 页面频繁刷新触发 GitHub API 限流。
func CheckLatestVersion() (latest string, hasUpdate bool, err error) {
	versionCheckMu.Lock()
	defer versionCheckMu.Unlock()

	ttl := versionCheckCacheTTL
	if versionCheckErr != nil {
		ttl = versionCheckErrorTTL
	}
	if !versionCheckAt.IsZero() && time.Since(versionCheckAt) < ttl {
		if versionCheckErr != nil {
			return "", false, versionCheckErr
		}
		return versionCheckLatest, compareVersion(version.VERSION, versionCheckLatest) < 0, nil
	}

	latest, err = fetchLatestVersion()
	if err != nil {
		versionCheckLatest = ""
		versionCheckErr = err
		versionCheckAt = time.Now()
		return "", false, err
	}

	versionCheckLatest = latest
	versionCheckErr = nil
	versionCheckAt = time.Now()
	return latest, compareVersion(version.VERSION, latest) < 0, nil
}
