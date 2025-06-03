// server/framework/logger.go
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"server/framework/config"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// 日志配置
	GlobalConfig = struct {
		Log struct {
			Level      string
			Filename   string
			MaxSize    int
			MaxBackups int
			MaxAge     int
			Compress   bool
		}
	}{}
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

	// 设置日志输出和级别
	hlog.SetOutput(writer)
	hlog.SetLevel(getLogLevel(config.GlobalConfig.Log.Level))
}

// LogRequest 记录请求日志
func LogRequest(serviceName, method string, req interface{}) {
	hlog.Infof("[%s] [%s] [%s] 收到请求: %+v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		req,
	)
}

// LogResponse 记录响应日志
func LogResponse(serviceName, method string, resp interface{}) {
	hlog.Infof("[%s] [%s] [%s] 返回响应: %+v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		resp,
	)
}

func LogDebug(serviceName, method string, msg string) {
	hlog.Debugf("[%s] [%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		msg,
	)
}

func LogInfo(serviceName, method string, msg string) {
	hlog.Infof("[%s] [%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		msg,
	)
}

func LogWarning(serviceName, method string, msg string) {
	hlog.Warnf("[%s] [%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		msg,
	)
}

func LogError(serviceName, method string, err error) {
	hlog.Errorf("[%s] [%s] [%s] 发生错误: %v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		err,
	)
}

func LogFatal(serviceName, method string, err error) {
	hlog.Fatalf("[%s] [%s] [%s] 发生错误: %v",
		time.Now().Format("2006-01-02 15:04:05.000"),
		serviceName,
		method,
		err,
	)
}

// getLogLevel 获取日志级别
func getLogLevel(level string) hlog.Level {
	switch level {
	case "debug":
		return hlog.LevelDebug
	case "info":
		return hlog.LevelInfo
	case "warn":
		return hlog.LevelWarn
	case "error":
		return hlog.LevelError
	default:
		return hlog.LevelInfo
	}
}
