package location

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RouteStorage 路线存储
type RouteStorage struct {
	storagePath string
}

// NewRouteStorage 创建路线存储
func NewRouteStorage() *RouteStorage {

	storagePath := filepath.Join("data", "routes.json")

	os.MkdirAll(filepath.Dir(storagePath), 0755)

	return &RouteStorage{
		storagePath: storagePath,
	}
}

// SaveRoutes 保存路线
func (rs *RouteStorage) SaveRoutes(routes []Route) error {
	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化路线失败: %w", err)
	}

	return os.WriteFile(rs.storagePath, data, 0644)
}

// LoadRoutes 加载路线
func (rs *RouteStorage) LoadRoutes() ([]Route, error) {
	data, err := os.ReadFile(rs.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Route{}, nil
		}
		return nil, fmt.Errorf("读取路线文件失败: %w", err)
	}

	var routes []Route
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("解析路线数据失败: %w", err)
	}

	return routes, nil
}
