package services

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type LogEntry struct {
	Level   string `json:"level"`
	Module  string `json:"module"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

const maxLogEntries = 1000

var Log = &LoggerService{logs: make([]LogEntry, 0, maxLogEntries)}
var appShuttingDown atomic.Bool

type LoggerService struct {
	mu   sync.RWMutex
	logs []LogEntry
}

func NewLoggerService() *LoggerService {
	return Log
}

// SetAppShuttingDown 设置应用是否处于退出阶段。
// 退出阶段会停止向前端发送 log-event，避免窗口销毁期间的事件分发死锁。
func SetAppShuttingDown(v bool) {
	appShuttingDown.Store(v)
}

// logMessage 记录日志消息
func (l *LoggerService) logMessage(level string, module string, message string) {
	entry := LogEntry{
		Level:   level,
		Module:  module,
		Message: message,
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	}
	l.mu.Lock()
	if len(l.logs) == maxLogEntries {
		copy(l.logs, l.logs[1:])
		l.logs[len(l.logs)-1] = entry
	} else {
		l.logs = append(l.logs, entry)
	}
	l.mu.Unlock()
	switch level {
	case "debug":
		slog.Debug(message, "module", module)
	case "info":
		slog.Info(message, "module", module)
	case "warn":
		slog.Warn(message, "module", module)
	case "error":
		slog.Error(message, "module", module)
	}

	// 退出阶段不再向前端分发日志事件，避免关闭流程阻塞。
	emitAppEvent("log-event", entry)
}

// GetLogs 获取所有日志
func (l *LoggerService) GetLogs() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]LogEntry{}, l.logs...)
}

func emitAppEvent(name string, data any) {
	if appShuttingDown.Load() {
		return
	}
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

// Debug 调试日志
func (l *LoggerService) Debug(module string, message string) {
	l.logMessage("debug", module, message)
}

// Info 信息日志
func (l *LoggerService) Info(module string, message string) {
	l.logMessage("info", module, message)
}

// Warn 警告日志
func (l *LoggerService) Warn(module string, message string) {
	l.logMessage("warn", module, message)
}

// Error 错误日志
func (l *LoggerService) Error(module string, message string) {
	l.logMessage("error", module, message)
}
