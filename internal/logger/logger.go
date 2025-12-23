package logger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogEntry 日志条目
type LogEntry struct {
	Time    string   `json:"time"`
	Level   LogLevel `json:"level"`
	Module  string   `json:"module"`
	Message string   `json:"message"`
}

// Service 日志服务
type Service struct {
	mu       sync.RWMutex
	logs     []LogEntry
	maxLogs  int
	wailsCtx context.Context
}

// NewService 创建日志服务
func NewService() *Service {
	return &Service{
		logs:    make([]LogEntry, 0, 500),
		maxLogs: 500,
	}
}

// Startup Wails启动回调
func (s *Service) Startup(ctx context.Context) {
	s.wailsCtx = ctx
	s.Info("系统", "日志服务已启动")
}

// log 内部日志方法
func (s *Service) log(level LogLevel, module, message string) {
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05.000"),
		Level:   level,
		Module:  module,
		Message: message,
	}

	s.mu.Lock()
	// 保持日志数量在限制内
	if len(s.logs) >= s.maxLogs {
		s.logs = s.logs[1:]
	}
	s.logs = append(s.logs, entry)
	s.mu.Unlock()

	// 发送到前端
	if s.wailsCtx != nil {
		runtime.EventsEmit(s.wailsCtx, "log:entry", entry)
	}

	// 同时输出到控制台
	fmt.Printf("[%s] [%s] [%s] %s\n", entry.Time, entry.Level, entry.Module, entry.Message)
}

// Debug 调试日志
func (s *Service) Debug(module, format string, args ...interface{}) {
	s.log(LevelDebug, module, fmt.Sprintf(format, args...))
}

// Info 信息日志
func (s *Service) Info(module, format string, args ...interface{}) {
	s.log(LevelInfo, module, fmt.Sprintf(format, args...))
}

// Warn 警告日志
func (s *Service) Warn(module, format string, args ...interface{}) {
	s.log(LevelWarn, module, fmt.Sprintf(format, args...))
}

// Error 错误日志
func (s *Service) Error(module, format string, args ...interface{}) {
	s.log(LevelError, module, fmt.Sprintf(format, args...))
}

// GetLogs 获取所有日志
func (s *Service) GetLogs() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]LogEntry, len(s.logs))
	copy(result, s.logs)
	return result
}

// ClearLogs 清除日志
func (s *Service) ClearLogs() {
	s.mu.Lock()
	s.logs = make([]LogEntry, 0, 500)
	s.mu.Unlock()
}
