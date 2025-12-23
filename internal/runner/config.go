package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigStorage 配置存储
type ConfigStorage struct {
	storagePath string
}

// NewConfigStorage 创建配置存储
func NewConfigStorage() *ConfigStorage {

	storagePath := filepath.Join("data", "config.json")
	os.MkdirAll(filepath.Dir(storagePath), 0755)

	return &ConfigStorage{
		storagePath: storagePath,
	}
}

// SaveConfig 保存配置
func (cs *ConfigStorage) SaveConfig(config RunConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return os.WriteFile(cs.storagePath, data, 0644)
}

// LoadConfig 加载配置
func (cs *ConfigStorage) LoadConfig() (*RunConfig, error) {
	data, err := os.ReadFile(cs.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 返回默认配置
			return &RunConfig{
				Speed:          8.0,
				SpeedVariation: 1.0,
				RouteVariation: 3.0,
				LoopCount:      1,
				UpdateInterval: 1000,
			}, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置数据失败: %w", err)
	}

	return &config, nil
}
