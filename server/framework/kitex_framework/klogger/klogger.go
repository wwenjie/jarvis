package klogger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"server/framework/config"

	"github.com/cloudwego/kitex/pkg/klog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger 初始化日志配置
func InitLogger() {
	// 创建日志目录
	if err := os.MkdirAll(filepath.Dir(config.GlobalConfig.Log.Filename), 0755); err != nil {
		panic(fmt.Sprintf("创建日志目录失败: %v", err))
	}

	// 设置日志输出
	writer := &lumberjack.Logger{
		Filename:   config.GlobalConfig.Log.Filename,
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	klog.SetOutput(writer)
	klog.SetLevel(getLogLevel(config.GlobalConfig.Log.Level))
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
