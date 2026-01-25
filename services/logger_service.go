package services

import (
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type LogEntry struct {
	Level   string `json:"level"`
	Module  string `json:"module"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

var Log *LoggerService

type LoggerService struct {
	logs []LogEntry
}

func NewLoggerService() *LoggerService {
	Log = &LoggerService{
		logs: make([]LogEntry, 0),
	}
	return Log
}

// logMessage 记录日志消息
func (l *LoggerService) logMessage(level string, module string, message string) {
	entry := LogEntry{
		Level:   level,
		Module:  module,
		Message: message,
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	}
	l.logs = append(l.logs, entry)
	if len(l.logs) > 1000 {
		l.logs = l.logs[1:]
	}
	fmt.Printf("[%s] [%s] [%s] %s\n", entry.Time, level, module, message)

	// 发送事件到前端
	application.Get().Event.Emit("log-event", entry)
}

// GetLogs 获取所有日志
func (l *LoggerService) GetLogs() []LogEntry {
	return l.logs
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
