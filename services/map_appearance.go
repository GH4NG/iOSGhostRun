package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const appearanceFileName = "map-appearance.json"
const maxAppearanceFileSize = 16 << 10

var mapStyleIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

// MapAppearance 只包含可迁移的外观偏好，不包含高德 Key 或安全密钥。
type MapAppearance struct {
	Version       int    `json:"version"`
	DarkStyle     string `json:"darkStyle"`
	CustomStyleID string `json:"customStyleId"`
	ThemeMode     string `json:"themeMode"`
}

func defaultMapAppearance() MapAppearance {
	return MapAppearance{Version: 1, DarkStyle: "darkblue", ThemeMode: "system"}
}

func (s *SettingsService) GetMapAppearance() (MapAppearance, error) {
	s.appearanceMu.Lock()
	defer s.appearanceMu.Unlock()
	appearance, err := readMapAppearanceFile()
	if errors.Is(err, os.ErrNotExist) {
		// 版本 0 告诉前端尝试迁移旧版 WebView 本地存储。
		appearance = defaultMapAppearance()
		appearance.Version = 0
		return appearance, nil
	}
	return appearance, err
}

func readMapAppearanceFile() (MapAppearance, error) {
	data, err := os.ReadFile(ResolveAppDir(appearanceFileName))
	if err != nil {
		return MapAppearance{}, err
	}
	return decodeMapAppearance(data)
}

func decodeMapAppearance(data []byte) (MapAppearance, error) {
	if len(data) > maxAppearanceFileSize {
		return MapAppearance{}, fmt.Errorf("地图外观文件过大")
	}
	var appearance MapAppearance
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&appearance); err != nil {
		return MapAppearance{}, fmt.Errorf("地图外观文件格式无效: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return MapAppearance{}, fmt.Errorf("地图外观文件只能包含一个 JSON 对象")
	}
	return validateMapAppearance(appearance)
}

func validateMapAppearance(appearance MapAppearance) (MapAppearance, error) {
	if appearance.Version != 1 {
		return MapAppearance{}, fmt.Errorf("不支持的地图外观文件版本")
	}
	appearance.CustomStyleID = strings.TrimSpace(appearance.CustomStyleID)
	appearance.CustomStyleID = strings.TrimPrefix(appearance.CustomStyleID, "amap://styles/")
	if appearance.CustomStyleID != "" && !mapStyleIDPattern.MatchString(appearance.CustomStyleID) {
		return MapAppearance{}, fmt.Errorf("自定义地图样式 ID 格式无效")
	}
	switch appearance.DarkStyle {
	case "darkblue", "blue", "dark":
	case "custom":
		if appearance.CustomStyleID == "" {
			return MapAppearance{}, fmt.Errorf("选择自定义深色地图时必须填写样式 ID")
		}
	default:
		return MapAppearance{}, fmt.Errorf("不支持的深色地图样式")
	}
	if appearance.ThemeMode != "system" && appearance.ThemeMode != "light" && appearance.ThemeMode != "dark" {
		return MapAppearance{}, fmt.Errorf("不支持的界面主题")
	}
	return appearance, nil
}

func (s *SettingsService) SaveMapAppearance(appearance MapAppearance) (MapAppearance, error) {
	s.appearanceMu.Lock()
	defer s.appearanceMu.Unlock()
	return saveMapAppearanceFile(appearance)
}

func saveMapAppearanceFile(appearance MapAppearance) (MapAppearance, error) {
	appearance, err := validateMapAppearance(appearance)
	if err != nil {
		return MapAppearance{}, err
	}
	data, err := json.MarshalIndent(appearance, "", "  ")
	if err != nil {
		return MapAppearance{}, err
	}
	if err := writeFileAtomically(ResolveAppDir(appearanceFileName), append(data, '\n'), 0600); err != nil {
		return MapAppearance{}, fmt.Errorf("保存地图外观失败: %w", err)
	}
	return appearance, nil
}

func (s *SettingsService) ImportMapAppearance() (MapAppearance, error) {
	app := application.Get()
	if app == nil {
		return MapAppearance{}, fmt.Errorf("请在桌面程序中导入外观文件")
	}
	path, err := app.Dialog.OpenFile().SetTitle("导入地图外观").AddFilter("地图外观 JSON", "*.json").PromptForSingleSelection()
	if err != nil || path == "" {
		return MapAppearance{}, err
	}
	appearance, err := importMapAppearanceFromPath(path)
	if err != nil {
		return MapAppearance{}, err
	}
	return s.SaveMapAppearance(appearance)
}

func importMapAppearanceFromPath(path string) (MapAppearance, error) {
	file, err := os.Open(path)
	if err != nil {
		return MapAppearance{}, fmt.Errorf("读取外观文件失败: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAppearanceFileSize+1))
	if err != nil {
		return MapAppearance{}, fmt.Errorf("读取外观文件失败: %w", err)
	}
	return decodeMapAppearance(data)
}

func (s *SettingsService) ExportMapAppearance() (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("请在桌面程序中导出外观文件")
	}
	s.appearanceMu.Lock()
	appearance, err := readMapAppearanceFile()
	s.appearanceMu.Unlock()
	if errors.Is(err, os.ErrNotExist) {
		appearance = defaultMapAppearance()
	} else if err != nil {
		return "", fmt.Errorf("读取地图外观失败: %w", err)
	}
	path, err := app.Dialog.SaveFile().SetFilename("iosghostrun-map-appearance.json").AddFilter("地图外观 JSON", "*.json").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if err := exportMapAppearanceToPath(path, appearance); err != nil {
		return "", err
	}
	return path, nil
}

func exportMapAppearanceToPath(path string, appearance MapAppearance) error {
	data, err := json.MarshalIndent(appearance, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomically(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("导出地图外观失败: %w", err)
	}
	return nil
}

// 高德要求样式 ID 与 JS API Key 属于同一账号。先确认样式内容可读，再交给地图 SDK。
func (s *SettingsService) CheckMapStyle(styleID string) (bool, error) {
	if !mapStyleIDPattern.MatchString(styleID) {
		return false, fmt.Errorf("自定义地图样式 ID 格式无效")
	}
	settings, err := s.readMapSettings()
	if err != nil || settings.Key == "" || settings.SecurityJsCode == "" {
		return false, fmt.Errorf("请先配置高德 JS API Key 和安全密钥")
	}
	query := url.Values{"styleid": {styleID}, "s": {"rsv3"}, "key": {settings.Key}, "jscode": {settings.SecurityJsCode}}
	requestURL := "https://webapi.amap.com/v4/map/styles2?" + query.Encode()
	client := newDownloadHTTPClient(10 * time.Second)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	defer client.CloseIdleConnections()
	response, err := client.Get(requestURL)
	if err != nil {
		return false, fmt.Errorf("无法连接高德样式服务，请检查网络")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("高德样式服务返回 HTTP %d", response.StatusCode)
	}
	var result struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return false, fmt.Errorf("高德样式服务响应无效")
	}
	return result.Data.Content != "", nil
}
