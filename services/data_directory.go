package services

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const dataFolderName = "iOSGhostRun-data"

var dataDirectory string

func InitializeDataDirectory() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取程序路径失败: %w", err)
	}
	var config string
	if runtime.GOOS == "darwin" {
		config, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("获取用户配置目录失败: %w", err)
		}
	}
	root := dataDirectoryPath(runtime.GOOS, exe, os.Getenv("APPIMAGE"), config)
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", fmt.Errorf("数据目录 %s 不可写，请将程序移到可写目录: %w", root, err)
	}
	if err := verifyWritable(root); err != nil {
		return "", fmt.Errorf("数据目录 %s 不可写，请将程序移到可写目录: %w", root, err)
	}
	for _, name := range []string{"pairrecords", "devimages"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			return "", fmt.Errorf("创建数据子目录 %s 失败: %w", name, err)
		}
	}
	if runtime.GOOS == "windows" {
		if err := os.MkdirAll(filepath.Join(root, "webview"), 0700); err != nil {
			return "", fmt.Errorf("创建 WebView 数据目录失败: %w", err)
		}
	}
	if runtime.GOOS == "linux" {
		for key, name := range map[string]string{
			"XDG_CONFIG_HOME": "config",
			"XDG_DATA_HOME":   "share",
			"XDG_CACHE_HOME":  "cache",
		} {
			path := filepath.Join(root, "webview", name)
			if err := os.MkdirAll(path, 0700); err != nil {
				return "", fmt.Errorf("创建 WebKit 数据目录失败: %w", err)
			}
			if err := os.Setenv(key, path); err != nil {
				return "", fmt.Errorf("配置 WebKit 数据目录失败: %w", err)
			}
		}
	}
	dataDirectory = root
	return root, nil
}

func dataDirectoryPath(goos, exe, appimage, config string) string {
	if goos == "darwin" {
		return filepath.Join(config, "iosghostrun")
	}
	if goos == "linux" && strings.HasPrefix(appimage, "/") {
		return filepath.Join(filepath.Dir(appimage), dataFolderName)
	}
	return filepath.Join(filepath.Dir(exe), dataFolderName)
}

func verifyWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".write-check-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}
