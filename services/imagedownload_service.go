package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/imagemounter"
)

// ImageDownloadService 镜像下载服务
type ImageDownloadService struct{}

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
	xcode15_4_ddi = "ddi-15F31d"
)

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *ImageDownloadService) downloadPersonalizedImage(basedir string) (string, error) {
	extractedPath := filepath.Join(basedir, xcode15_4_ddi)
	restorePath := filepath.Join(extractedPath, "Restore")

	if fileExists(restorePath) {
		Log.Info("ImageDownloadService", fmt.Sprintf("使用已下载的镜像: %s", restorePath))
		return restorePath, nil
	}

	downloadURL := deviceboxURL + xcode15_4_ddi + ".zip"
	zipPath := filepath.Join(basedir, xcode15_4_ddi+".zip")

	Log.Info("ImageDownloadService", fmt.Sprintf("下载 iOS 17+ 镜像: %s", downloadURL))
	if err := s.downloadFile(zipPath, downloadURL); err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}

	Log.Info("ImageDownloadService", "解压镜像...")
	if _, _, err := ios.Unzip(zipPath, extractedPath); err != nil {
		return "", fmt.Errorf("解压失败: %w", err)
	}

	return restorePath, nil
}

func (s *ImageDownloadService) downloadDeveloperImage(basedir string, productVersion string) (string, error) {
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

		if err := s.downloadFile(imagePath, mirrorImageURL); err != nil {
			lastErr = err
			Log.Warn("ImageDownloadService", fmt.Sprintf("下载镜像失败: %v, 尝试下一个镜像...", err))
			continue
		}

		if err := s.downloadFile(signaturePath, mirrorSignatureURL); err != nil {
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

// downloadFile 下载文件到本地
func (s *ImageDownloadService) downloadFile(filepathLocal string, url string) error {
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
