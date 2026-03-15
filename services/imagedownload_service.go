package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielpaulus/go-ios/ios/imagemounter"
	"howett.net/plist"
)

type BuildManifestInfo struct {
	PersonalizedDMGPath string
	TrustCachePath      string
}

type buildManifest struct {
	BuildIdentities []buildIdentity
}

type buildIdentity struct {
	BoardID  string `plist:"ApBoardID"`
	ChipID   string `plist:"ApChipID"`
	Manifest struct {
		LoadableTrustCache struct {
			Digest []byte
			Info   struct {
				Path string
			}
		}
		PersonalizedDmg struct {
			Digest []byte
			Info   struct {
				Path string
			}
		} `plist:"PersonalizedDMG"`
	}
}

var (
	githubMirrors = []string{
		"",
		"https://ghfast.top/",
		"https://gh-proxy.com/",
		"https://mirror.ghproxy.com/",
	}

	deviceboxURL  = "https://deviceboxhq.com/"
	imageFile     = "DeveloperDiskImage.dmg"
	signatureFile = "DeveloperDiskImage.dmg.signature"

	githubDDIBaseURL = "https://github.com/doronz88/DeveloperDiskImage/raw/main/PersonalizedImages/Xcode_iOS_DDI_Personalized"
	ddiManifestFile  = "BuildManifest.plist"
)

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func downloadPersonalizedImage(basedir string) (string, error) {
	extractedPath := filepath.Join(basedir, "Xcode_iOS_DDI_Personalized")

	manifestPath := filepath.Join(extractedPath, ddiManifestFile)
	if !fileExists(manifestPath) {
		if err := os.MkdirAll(extractedPath, 0o755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}

		// 1. 先下载 BuildManifest.plist
		Log.Info("ImageDownloadService", "下载 BuildManifest.plist")
		manifestURL := githubDDIBaseURL + "/" + ddiManifestFile
		var lastErr error
		for _, mirror := range githubMirrors {
			var mirrorURL string
			if mirror == "" {
				mirrorURL = manifestURL
				Log.Info("ImageDownloadService", "尝试从 GitHub 原始地址下载...")
			} else {
				mirrorURL = mirror + manifestURL
				Log.Info("ImageDownloadService", fmt.Sprintf("尝试从镜像站下载: %s", mirror))
			}
			if err := downloadFile(manifestPath, mirrorURL); err != nil {
				lastErr = err
				Log.Warn("ImageDownloadService", fmt.Sprintf("下载 BuildManifest.plist 失败: %v, 尝试下一个镜像...", err))
				continue
			}
			Log.Info("ImageDownloadService", "BuildManifest.plist 下载成功")
			break
		}
		if lastErr != nil && !fileExists(manifestPath) {
			return "", fmt.Errorf("下载 BuildManifest.plist 失败: %w", lastErr)
		}
	} else {
		Log.Info("ImageDownloadService", "BuildManifest.plist 已存在，跳过下载")
	}

	// 2. 解析 Manifest 获取文件路径
	Log.Info("ImageDownloadService", "解析 BuildManifest.plist")
	manifestInfo, err := parseManifestPaths(manifestPath)
	if err != nil {
		return "", fmt.Errorf("解析 manifest 失败: %w", err)
	}

	dmgPath := filepath.Join(extractedPath, manifestInfo.PersonalizedDMGPath)
	trustCachePath := filepath.Join(extractedPath, manifestInfo.TrustCachePath)

	if fileExists(dmgPath) && fileExists(trustCachePath) {
		Log.Info("ImageDownloadService", fmt.Sprintf("使用已下载的镜像: %s", dmgPath))
		return extractedPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(dmgPath), 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(trustCachePath), 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	dmgURL := githubDDIBaseURL + "/Image.dmg"
	trustCacheURL := githubDDIBaseURL + "/Image.dmg.trustcache"
	var lastErr error
	for _, mirror := range githubMirrors {
		var mirrorDmgURL, mirrorTrustCacheURL string
		if mirror == "" {
			mirrorDmgURL = dmgURL
			mirrorTrustCacheURL = trustCacheURL
			Log.Info("ImageDownloadService", "尝试从 GitHub 原始地址下载...")
		} else {
			mirrorDmgURL = mirror + dmgURL
			mirrorTrustCacheURL = mirror + trustCacheURL
			Log.Info("ImageDownloadService", fmt.Sprintf("尝试从镜像站下载: %s", mirror))
		}

		if err := downloadFile(dmgPath, mirrorDmgURL); err != nil {
			lastErr = err
			Log.Warn("ImageDownloadService", fmt.Sprintf("下载镜像失败: %v, 尝试下一个镜像...", err))
			continue
		}

		if err := downloadFile(trustCachePath, mirrorTrustCacheURL); err != nil {
			lastErr = err
			os.Remove(dmgPath)
			Log.Warn("ImageDownloadService", fmt.Sprintf("下载 TrustCache 失败: %v, 尝试下一个镜像...", err))
			continue
		}

		Log.Info("ImageDownloadService", "Personalized DDI 下载完成")
		return extractedPath, nil
	}
	if lastErr != nil {
		return "", fmt.Errorf("所有镜像下载失败: %w", lastErr)
	}
	return "", fmt.Errorf("所有镜像下载失败")
}

func downloadDeveloperImage(basedir string, productVersion string) (string, error) {
	version := imagemounter.MatchAvailable(productVersion)
	versionDir := strings.Split(version, " (")[0]
	imageDir := filepath.Join(basedir, versionDir)
	imagePath := filepath.Join(imageDir, imageFile)
	signaturePath := filepath.Join(imageDir, signatureFile)

	// 如果两个文件都已存在，直接使用
	if fileExists(imagePath) && fileExists(signaturePath) {
		Log.Info("ImageDownloadService", fmt.Sprintf("使用已下载的镜像: %s", imagePath))
		return imagePath, nil
	}

	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	baseURL := fmt.Sprintf("https://github.com/mspvirajpatel/Xcode_Developer_Disk_Images/raw/master/Developer%%20Disk%%20Image/%s", version)
	imageURL := baseURL + "/" + imageFile
	signatureURL := baseURL + "/" + signatureFile

	var lastErr error
	for _, mirror := range githubMirrors {
		mirrorImageURL := mirror + imageURL
		mirrorSignatureURL := mirror + signatureURL

		if mirror == "" {
			Log.Info("ImageDownloadService", "尝试从 GitHub 原始地址下载...")
		} else {
			Log.Info("ImageDownloadService", fmt.Sprintf("尝试从镜像站下载: %s", mirror))
		}

		if err := downloadFile(imagePath, mirrorImageURL); err != nil {
			lastErr = err
			Log.Warn("ImageDownloadService", fmt.Sprintf("下载镜像失败: %v, 尝试下一个镜像...", err))
			continue
		}

		if err := downloadFile(signaturePath, mirrorSignatureURL); err != nil {
			lastErr = err
			os.Remove(imagePath)
			Log.Warn("ImageDownloadService", fmt.Sprintf("下载签名失败: %v, 尝试下一个镜像...", err))
			continue
		}

		Log.Info("ImageDownloadService", "下载成功")
		return imagePath, nil
	}

	return "", fmt.Errorf("所有镜像下载失败: %w", lastErr)
}

func parseManifestPaths(manifestPath string) (*BuildManifestInfo, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("打开 manifest 文件失败: %w", err)
	}
	defer file.Close()

	decoder := plist.NewDecoder(file)
	var manifest buildManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("解析 manifest 文件失败: %w", err)
	}

	if len(manifest.BuildIdentities) == 0 {
		return nil, fmt.Errorf("manifest 中没有 BuildIdentities")
	}

	// 使用第一个身份的信息
	identity := manifest.BuildIdentities[0]
	dmgPath := identity.Manifest.PersonalizedDmg.Info.Path
	trustCachePath := identity.Manifest.LoadableTrustCache.Info.Path

	if dmgPath == "" {
		return nil, fmt.Errorf("未能从 manifest 中获取 PersonalizedDMG 路径")
	}
	if trustCachePath == "" {
		return nil, fmt.Errorf("未能从 manifest 中获取 LoadableTrustCache 路径")
	}

	Log.Info("ImageDownloadService", fmt.Sprintf("解析路径: DMG=%s, TrustCache=%s", dmgPath, trustCachePath))

	return &BuildManifestInfo{
		PersonalizedDMGPath: dmgPath,
		TrustCachePath:      trustCachePath,
	}, nil
}

// downloadFile 下载文件到本地
func downloadFile(filepathLocal string, url string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(filepathLocal)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
