package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
	updatergithub "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// updateManifestURL 更新清单地址
const updateManifestURL = "https://github.com/GH4NG/iOSGhostRun/releases/latest/download/update.json"

type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseNotes   string `json:"releaseNotes"`
	PublishedAt    string `json:"publishedAt"`
	AssetName      string `json:"assetName"`
	AssetSize      int64  `json:"assetSize"`
}

// UpdateService keeps the existing frontend API while delegating update
// discovery, download, verification, extraction and restart to Wails.
type UpdateService struct {
	mu      sync.Mutex
	engine  *updater.Updater
	pending *updater.Release
}

func NewUpdateService() *UpdateService {
	return &UpdateService{}
}

// ConfigureUpdateService connects the service to the updater owned by the
// Wails application. It is a package function, so it is not a frontend RPC.
func ConfigureUpdateService(service *UpdateService, engine *updater.Updater, currentVersion string) error {
	if service == nil || engine == nil {
		return errors.New("初始化更新服务失败")
	}
	updateHTTPClient := newDownloadHTTPClient(30 * time.Second)
	provider, err := endpoint.New(endpoint.Config{
		URL:        updateManifestURL,
		Channel:    "stable",
		HTTPClient: updateHTTPClient,
	})
	if err != nil {
		return fmt.Errorf("创建更新源失败: %w", err)
	}
	githubProvider, err := updatergithub.New(updatergithub.Config{
		Repository:    "GH4NG/iOSGhostRun",
		ChecksumAsset: "SHA256SUMS",
		HTTPClient:    updateHTTPClient,
	})
	if err != nil {
		return fmt.Errorf("创建 GitHub 更新源失败: %w", err)
	}
	if err := engine.Init(updater.Config{
		CurrentVersion: currentVersion,
		Providers:      []updater.Provider{provider, githubProvider},
		Window:         updater.WindowNone,
	}); err != nil {
		return fmt.Errorf("初始化 Wails Updater 失败: %w", err)
	}

	service.mu.Lock()
	service.engine = engine
	service.mu.Unlock()
	return nil
}

func (s *UpdateService) CheckForUpdate() (UpdateInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine == nil {
		return UpdateInfo{}, errors.New("更新服务尚未初始化")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	release, err := s.engine.Check(ctx)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("检查更新失败: %w", err)
	}
	if release == nil {
		s.pending = nil
		return UpdateInfo{
			Available:      false,
			CurrentVersion: s.engine.CurrentVersion(),
			LatestVersion:  s.engine.CurrentVersion(),
		}, nil
	}

	s.pending = release
	return updateInfoFromRelease(s.engine.CurrentVersion(), release), nil
}

// DownloadAndInstall lets Wails download, verify and stage the pending
// release, then uses the built-in updater helper to restart into it.
func (s *UpdateService) DownloadAndInstall(version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine == nil {
		return errors.New("更新服务尚未初始化")
	}
	if s.pending == nil {
		return errors.New("没有待安装的更新，请重新检查")
	}
	if strings.TrimPrefix(strings.TrimSpace(version), "v") != strings.TrimPrefix(s.pending.Version, "v") {
		return errors.New("发布版本已变化，请重新检查更新")
	}
	if runtime.GOOS == "linux" && os.Getenv("APPIMAGE") != "" {
		return errors.New("当前 Wails 版本无法替换运行中的 AppImage，请从 GitHub Releases 下载新版本并覆盖原 AppImage")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := s.engine.DownloadAndInstall(ctx); err != nil {
		return fmt.Errorf("下载或验证更新失败: %w", err)
	}
	if err := s.engine.Restart(context.Background()); err != nil {
		return fmt.Errorf("启动更新程序失败: %w", err)
	}
	return nil
}

func updateInfoFromRelease(current string, release *updater.Release) UpdateInfo {
	info := UpdateInfo{
		Available:      true,
		CurrentVersion: current,
		LatestVersion:  release.Version,
		ReleaseNotes:   release.Notes,
		AssetName:      release.Artifact.Filename,
		AssetSize:      release.Artifact.Size,
	}
	if !release.PublishedAt.IsZero() {
		info.PublishedAt = release.PublishedAt.Format(time.RFC3339)
	}
	return info
}
