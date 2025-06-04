package klogger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"server/framework/config"

	"github.com/cloudwego/kitex/pkg/klog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger 初始化日志配置
func InitLogger() {
	// 创建日志目录
	logDir := filepath.Join(config.GlobalConfig.Log.LogPath, config.GlobalConfig.Service.Name)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("创建日志目录失败: %v", err))
	}

	// 设置不同级别的日志输出
	debugWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "debug.log"),
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	infoWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "info.log"),
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	warningWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "warning.log"),
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	errorWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "error.log"),
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	// 设置日志输出和级别
	klog.SetLevel(getLogLevel(config.GlobalConfig.Log.Level))

	// 创建一个自定义的 writer，根据日志级别写入不同的文件
	levelWriter := &LevelWriter{
		debugWriter:   debugWriter,
		infoWriter:    infoWriter,
		warningWriter: warningWriter,
		errorWriter:   errorWriter,
	}

	// 设置日志输出
	klog.SetOutput(levelWriter)
}

// LevelWriter 根据日志级别写入不同的文件
type LevelWriter struct {
	debugWriter   io.Writer
	infoWriter    io.Writer
	warningWriter io.Writer
	errorWriter   io.Writer
}

// Write 实现 io.Writer 接口
func (w *LevelWriter) Write(p []byte) (n int, err error) {
	// 解析日志级别
	level := parseLogLevel(string(p))

	// 根据日志级别写入不同的文件
	switch level {
	case "DEBUG":
		return w.debugWriter.Write(p)
	case "INFO":
		return io.MultiWriter(w.debugWriter, w.infoWriter).Write(p)
	case "WARN":
		return io.MultiWriter(w.debugWriter, w.infoWriter, w.warningWriter).Write(p)
	case "ERROR":
		return io.MultiWriter(w.debugWriter, w.infoWriter, w.warningWriter, w.errorWriter).Write(p)
	default:
		return w.debugWriter.Write(p)
	}
}

// parseLogLevel 从日志内容中解析日志级别
func parseLogLevel(log string) string {
	if strings.Contains(log, "[DEBUG]") {
		return "DEBUG"
	}
	if strings.Contains(log, "[INFO]") {
		return "INFO"
	}
	if strings.Contains(log, "[WARN]") {
		return "WARN"
	}
	if strings.Contains(log, "[ERROR]") {
		return "ERROR"
	}
	return "DEBUG"
}

// LogRequest 记录请求日志
func LogRequest(serviceName, method string, req interface{}) {
	klog.Infof("[%s] [%s] [%s] 收到请求: %+v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		req,
	)
}

// LogResponse 记录响应日志
func LogResponse(serviceName, method string, resp interface{}) {
	klog.Infof("[%s] [%s] [%s] 返回响应: %+v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		resp,
	)
}

func LogDebug(serviceName, method string, msg string) {
	klog.Debugf("[%s] [%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		msg,
	)
}

func LogInfo(serviceName, method string, msg string) {
	klog.Infof("[%s] [%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		msg,
	)
}

func LogWarning(serviceName, method string, msg string) {
	klog.Warnf("[%s] [%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		msg,
	)
}

func LogError(serviceName, method string, err error) {
	klog.Errorf("[%s] [%s] [%s] 发生错误: %v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		err,
	)
}

func LogFatal(serviceName, method string, err error) {
	klog.Fatalf("[%s] [%s] [%s] 发生错误: %v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		err,
	)
}

// getLogLevel 获取日志级别
func getLogLevel(level string) klog.Level {
	switch level {
	case "debug":
		return klog.LevelDebug
	case "info":
		return klog.LevelInfo
	case "warn":
		return klog.LevelWarn
	case "error":
		return klog.LevelError
	default:
		return klog.LevelInfo
	}
}
