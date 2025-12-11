package logger

import (
	"os"
	"path/filepath"

	"github.com/cuihe500/astro/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var defaultLogger *zap.Logger

// Init 初始化日志系统
func Init(cfg *config.LogConfig) error {
	// 解析日志级别
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// 控制台编码器配置（人类可读格式）
	consoleEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder, // 彩色大写级别
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// JSON 文件编码器配置
	jsonEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写级别
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建输出核心
	var cores []zapcore.Core

	// 控制台输出（人类可读的优雅模式）
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
	cores = append(cores, consoleCore)

	// 文件输出（如果配置了文件路径）
	if cfg.File != "" {
		// 确保日志目录存在
		if dir := filepath.Dir(cfg.File); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		// 配置日志轮转
		maxSize := cfg.MaxSize
		if maxSize <= 0 {
			maxSize = 100 // 默认 100MB
		}
		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = 10 // 默认保留 10 个
		}
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 30 // 默认保留 30 天
		}

		writer := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   cfg.Compress,
		}

		fileEncoder := zapcore.NewJSONEncoder(jsonEncoderConfig)
		fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(writer), level)
		cores = append(cores, fileCore)
	}

	// 合并核心
	core := zapcore.NewTee(cores...)

	// 创建 logger
	defaultLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return nil
}

// Default 返回默认 Logger
func Default() *zap.Logger {
	if defaultLogger == nil {
		// 如果未初始化，返回开发模式 logger
		defaultLogger, _ = zap.NewDevelopment()
	}
	return defaultLogger
}

// Sugar 返回 SugaredLogger
func Sugar() *zap.SugaredLogger {
	return Default().Sugar()
}

// Sync 刷新日志缓冲
func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}

// Debug 输出调试日志
func Debug(msg string, fields ...zap.Field) {
	Default().Debug(msg, fields...)
}

// Info 输出信息日志
func Info(msg string, fields ...zap.Field) {
	Default().Info(msg, fields...)
}

// Warn 输出警告日志
func Warn(msg string, fields ...zap.Field) {
	Default().Warn(msg, fields...)
}

// Error 输出错误日志
func Error(msg string, fields ...zap.Field) {
	Default().Error(msg, fields...)
}

// Fatal 输出致命错误日志并退出
func Fatal(msg string, fields ...zap.Field) {
	Default().Fatal(msg, fields...)
}

// With 创建带有额外字段的 Logger
func With(fields ...zap.Field) *zap.Logger {
	return Default().With(fields...)
}
