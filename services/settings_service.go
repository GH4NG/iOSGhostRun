package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type MapSettings struct {
	Key            string `json:"key"`
	SecurityJsCode string `json:"securityJsCode"`
}

// 已保存的安全密钥不回传到前端；空输入可保留同一 Key 的原密钥。
type MapSettingsView struct {
	Key                    string `json:"key"`
	SecurityCodeConfigured bool   `json:"securityCodeConfigured"`
}

type MapRuntimeConfig struct {
	Key         string `json:"key"`
	ServiceHost string `json:"serviceHost"`
}

type SettingsService struct {
	mu             sync.Mutex
	appearanceMu   sync.Mutex
	runtimeMu      sync.Mutex
	wintunLicense  []byte
	proxyControlMu sync.Mutex
	proxyMu        sync.RWMutex
	mapProxy       *amapProxy
	proxyError     string
}

func NewSettingsService(wintunLicense []byte) *SettingsService {
	return &SettingsService{wintunLicense: wintunLicense}
}

func (s *SettingsService) GetMapSettings() (MapSettingsView, error) {
	settings, err := s.readMapSettings()
	return mapSettingsView(settings), err
}

func mapSettingsView(settings MapSettings) MapSettingsView {
	return MapSettingsView{Key: settings.Key, SecurityCodeConfigured: settings.SecurityJsCode != ""}
}

func (s *SettingsService) readMapSettings() (MapSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readMapSettingsFile()
}

func readMapSettingsFile() (MapSettings, error) {
	var settings MapSettings
	data, err := os.ReadFile(ResolveAppDir("map-settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, fmt.Errorf("读取地图设置失败: %w", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, fmt.Errorf("解析地图设置失败，请打开设置重新保存: %w", err)
	}
	return settings, nil
}

func (s *SettingsService) SaveMapSettings(settings MapSettings) (MapSettingsView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings.Key = strings.TrimSpace(settings.Key)
	settings.SecurityJsCode = strings.TrimSpace(settings.SecurityJsCode)
	if settings.Key != "" && settings.SecurityJsCode == "" {
		previous, err := readMapSettingsFile()
		if err == nil && previous.Key == settings.Key {
			settings.SecurityJsCode = previous.SecurityJsCode
		}
	}
	settings, err := validateMapSettings(settings)
	if err != nil {
		return MapSettingsView{}, err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return MapSettingsView{}, err
	}
	if err := writeFileAtomically(ResolveAppDir("map-settings.json"), data, 0600); err != nil {
		return MapSettingsView{}, fmt.Errorf("保存地图设置失败: %w", err)
	}
	s.proxyMu.RLock()
	p := s.mapProxy
	s.proxyMu.RUnlock()
	if p != nil {
		p.mu.Lock()
		p.lastError = ""
		p.mu.Unlock()
	}
	return mapSettingsView(settings), nil
}

func validateMapSettings(settings MapSettings) (MapSettings, error) {
	settings.Key = strings.TrimSpace(settings.Key)
	settings.SecurityJsCode = strings.TrimSpace(settings.SecurityJsCode)
	if settings.Key == "" && settings.SecurityJsCode == "" {
		return settings, nil
	}
	if settings.Key == "" || settings.SecurityJsCode == "" {
		return settings, fmt.Errorf("请同时填写 Web 端（JS API）Key 和安全密钥（securityJsCode）")
	}
	if len(settings.Key) > 256 || strings.ContainsAny(settings.Key, " \t\r\n") {
		return settings, fmt.Errorf("JS API Key 格式无效，请检查是否包含空格或换行")
	}
	if len(settings.SecurityJsCode) > 256 || strings.ContainsAny(settings.SecurityJsCode, " \t\r\n") || strings.Contains(settings.SecurityJsCode, "://") {
		return settings, fmt.Errorf("安全密钥格式无效，请填写高德生成的 securityJsCode，而非代理地址")
	}
	return settings, nil
}

// 在同一目录写临时文件后替换，避免中途退出破坏已有配置或运行库。
func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".iosghostrun-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
