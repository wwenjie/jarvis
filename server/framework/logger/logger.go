package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"server/framework/config"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// 全局 zap logger
	ZapLogger *zap.Logger
)

// getZapLevel 将配置的日志级别转换为 zap 的日志级别
func getZapLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// InitLogger 初始化日志配置
func InitLogger() {
	// 检查配置是否已加载
	if config.GlobalConfig == nil {
		panic("配置未加载，请先调用 config.LoadConfig")
	}

	fmt.Printf("开始初始化日志，配置信息：%+v\n", config.GlobalConfig.Log)

	// 创建日志目录
	logDir := filepath.Join(config.GlobalConfig.Log.LogPath, config.GlobalConfig.Service.Name)
	fmt.Printf("日志目录：%s\n", logDir)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("创建日志目录失败: %v", err))
	}

	// 配置 zap 的编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建不同级别的日志文件
	debugWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "debug.log"),
		MaxSize:    config.GlobalConfig.Log.MaxSize, // MB
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge, // 天
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

	// 获取配置的日志级别
	level := getZapLevel(config.GlobalConfig.Log.Level)
	fmt.Printf("日志级别：%v\n", level)

	// 创建 zap 的 core
	debugCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(debugWriter)),
		zapcore.DebugLevel, // debug 级别，会记录所有日志
	)

	infoCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(infoWriter)),
		zapcore.InfoLevel, // info 级别，会记录 info 及以上级别的日志
	)

	warnCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(warningWriter)),
		zapcore.WarnLevel, // warn 级别，会记录 warn 及以上级别的日志
	)

	errorCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(errorWriter)),
		zapcore.ErrorLevel, // error 级别，只记录 error 级别的日志
	)

	// 创建 zap logger
	ZapLogger = zap.New(
		zapcore.NewTee(debugCore, infoCore, warnCore, errorCore),
		zap.AddCaller(),
		zap.AddCallerSkip(1), // 跳过一层调用，用于我们自己的业务代码
	)

	// 设置 Hertz 的日志系统
	hlog.SetLogger(&HertzLogger{skip: 2}) // 为 Hertz 日志设置额外的 skip 值

	fmt.Printf("日志初始化完成\n")
}

// Debug 输出 debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Debug(msg, fields...)
}

// Info 输出 info 级别日志
func Info(msg string, fields ...zap.Field) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Info(msg, fields...)
}

// Warn 输出 warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Warn(msg, fields...)
}

// Error 输出 error 级别日志
func Error(msg string, fields ...zap.Field) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Error(msg, fields...)
}

// Fatal 输出 fatal 级别日志
func Fatal(msg string, fields ...zap.Field) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Fatal(msg, fields...)
}

// Debugf 输出带格式化的 debug 级别日志
func Debugf(format string, args ...interface{}) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Sugar().Debugf(format, args...)
}

// Infof 输出带格式化的 info 级别日志
func Infof(format string, args ...interface{}) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Sugar().Infof(format, args...)
}

// Warnf 输出带格式化的 warn 级别日志
func Warnf(format string, args ...interface{}) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Sugar().Warnf(format, args...)
}

// Errorf 输出带格式化的 error 级别日志
func Errorf(format string, args ...interface{}) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Sugar().Errorf(format, args...)
}

// Fatalf 输出带格式化的 fatal 级别日志
func Fatalf(format string, args ...interface{}) {
	if ZapLogger == nil {
		return
	}
	ZapLogger.Sugar().Fatalf(format, args...)
}

// HertzLogger 实现 Hertz 的日志接口
type HertzLogger struct {
	skip int // 额外的 skip 值
}

// getLogger 获取带有正确 skip 值的 logger
func (l *HertzLogger) getLogger() *zap.Logger {
	return ZapLogger.WithOptions(zap.AddCallerSkip(l.skip))
}

func (l *HertzLogger) Trace(v ...interface{}) {
	l.getLogger().Debug(fmt.Sprint(v...))
}

func (l *HertzLogger) Debug(v ...interface{}) {
	l.getLogger().Debug(fmt.Sprint(v...))
}

func (l *HertzLogger) Info(v ...interface{}) {
	l.getLogger().Info(fmt.Sprint(v...))
}

func (l *HertzLogger) Notice(v ...interface{}) {
	l.getLogger().Info(fmt.Sprint(v...))
}

func (l *HertzLogger) Warn(v ...interface{}) {
	l.getLogger().Warn(fmt.Sprint(v...))
}

func (l *HertzLogger) Error(v ...interface{}) {
	l.getLogger().Error(fmt.Sprint(v...))
}

func (l *HertzLogger) Fatal(v ...interface{}) {
	l.getLogger().Fatal(fmt.Sprint(v...))
}

func (l *HertzLogger) Tracef(format string, v ...interface{}) {
	l.getLogger().Sugar().Debugf(format, v...)
}

func (l *HertzLogger) Debugf(format string, v ...interface{}) {
	l.getLogger().Sugar().Debugf(format, v...)
}

func (l *HertzLogger) Infof(format string, v ...interface{}) {
	l.getLogger().Sugar().Infof(format, v...)
}

func (l *HertzLogger) Noticef(format string, v ...interface{}) {
	l.getLogger().Sugar().Infof(format, v...)
}

func (l *HertzLogger) Warnf(format string, v ...interface{}) {
	l.getLogger().Sugar().Warnf(format, v...)
}

func (l *HertzLogger) Errorf(format string, v ...interface{}) {
	l.getLogger().Sugar().Errorf(format, v...)
}

func (l *HertzLogger) Fatalf(format string, v ...interface{}) {
	l.getLogger().Sugar().Fatalf(format, v...)
}

// 实现带上下文的日志方法
func (l *HertzLogger) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Debugf(format, v...)
}

func (l *HertzLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Debugf(format, v...)
}

func (l *HertzLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Infof(format, v...)
}

func (l *HertzLogger) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Infof(format, v...)
}

func (l *HertzLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Warnf(format, v...)
}

func (l *HertzLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Errorf(format, v...)
}

func (l *HertzLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	l.getLogger().Sugar().Fatalf(format, v...)
}

// 实现其他必需的接口方法
func (l *HertzLogger) SetLevel(level hlog.Level) {
	// 不需要实现，因为我们使用 zap 的日志级别
}

func (l *HertzLogger) SetOutput(w io.Writer) {
	// 不需要实现，因为我们使用 zap 的输出
}

func (l *HertzLogger) SetOutputPath(path string) {
	// 不需要实现，因为我们使用 zap 的输出路径
}

func (l *HertzLogger) SetFormatter(formatter interface{}) {
	// 不需要实现，因为我们使用 zap 的格式化器
}

func (l *HertzLogger) SetReportCaller(reportCaller bool) {
	// 不需要实现，因为我们使用 zap 的调用者报告
}

func (l *HertzLogger) SetPrefix(prefix string) {
	// 不需要实现，因为我们使用 zap 的前缀
}

func (l *HertzLogger) SetFlags(flag int) {
	// 不需要实现，因为我们使用 zap 的标志
}

func (l *HertzLogger) SetDefaultLogger(logger hlog.FullLogger) {
	// 不需要实现，因为我们使用 zap 的默认日志记录器
}

func (l *HertzLogger) GetDefaultLogger() hlog.FullLogger {
	return l
}

func (l *HertzLogger) GetLogger() hlog.FullLogger {
	return l
}

func (l *HertzLogger) GetLevel() hlog.Level {
	return hlog.LevelDebug // 返回最低级别，让 zap 处理实际的日志级别
}

func (l *HertzLogger) GetOutput() io.Writer {
	return os.Stdout // 返回标准输出，让 zap 处理实际的输出
}

func (l *HertzLogger) GetFormatter() interface{} {
	return nil // 返回 nil，让 zap 处理实际的格式化
}

func (l *HertzLogger) GetReportCaller() bool {
	return true // 返回 true，让 zap 处理实际的调用者报告
}

func (l *HertzLogger) GetPrefix() string {
	return "" // 返回空字符串，让 zap 处理实际的前缀
}

func (l *HertzLogger) GetFlags() int {
	return 0 // 返回 0，让 zap 处理实际的标志
}

// GetHertzLogger 获取 Hertz 日志记录器实例
func GetHertzLogger() *HertzLogger {
	return &HertzLogger{skip: 2}
}
