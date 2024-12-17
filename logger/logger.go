// Package logger 提供一个灵活的日志记录器，支持多种输出形式和配置选项。
package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 定义日志接口
type Logger interface {
	Debug(msg string, fields ...zapcore.Field)
	Info(msg string, fields ...zapcore.Field)
	Warn(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	DPanic(msg string, fields ...zapcore.Field)
	Panic(msg string, fields ...zapcore.Field)
	Fatal(msg string, fields ...zapcore.Field)

	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	DPanicf(template string, args ...interface{})
	Panicf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})

	// 标准库 log 接口
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})

	Sync() error
	With(fields ...zapcore.Field) Logger
}

// Options 定义日志配置选项
type Options struct {
	// 基础配置
	Level    zapcore.Level
	Filename string
	Stdout   bool

	// 文件轮转配置
	Rotation RotationOptions

	// 扩展选项
	CallerSkip int
	Fields     []zap.Field // 初始字段
}

// RotationOptions 定义日志轮转配置
type RotationOptions struct {
	MaxSize    int  // MB
	MaxBackups int  // 文件个数
	MaxAge     int  // 天数
	Compress   bool // 是否压缩
	LocalTime  bool // 使用本地时间
}

// DefaultOptions 返回默认配置
func DefaultOptions() Options {
	return Options{
		Level:  zapcore.InfoLevel,
		Stdout: true,
		Rotation: RotationOptions{
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
			LocalTime:  true,
		},
		CallerSkip: 1,
	}
}

// logger 实现 Logger 接口
type logger struct {
	zap     *zap.Logger
	sugar   *zap.SugaredLogger
	stdLog  *log.Logger
	opts    Options
	isDebug bool
}

// New 创建新的日志实例并返回 Logger 接口。
func New(options Options) (Logger, error) {
	encodingConfig := &zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	cores, err := buildCores(options, encodingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build cores with options %+v: %w", options, err)
	}

	zapOpts := []zap.Option{zap.AddCaller()}
	if options.Level == zapcore.DebugLevel {
		zapOpts = append(zapOpts, zap.Development())
	}
	if options.CallerSkip > 0 {
		zapOpts = append(zapOpts, zap.AddCallerSkip(options.CallerSkip))
	}
	if len(options.Fields) > 0 {
		zapOpts = append(zapOpts, zap.Fields(options.Fields...))
	}

	l := &logger{
		zap:     zap.New(zapcore.NewTee(cores...), zapOpts...),
		opts:    options,
		isDebug: options.Level == zapcore.DebugLevel,
	}
	l.sugar = l.zap.Sugar()

	// 创建标准库logger
	writer := zapcore.AddSync(l)
	l.stdLog = log.New(writer, "", 0)

	return l, nil
}

// NewDefaultLogger 创建一个使用默认配置的日志实例并返回 Logger 接口。
func NewDefaultLogger() (Logger, error) {
	opts := DefaultOptions()
	return New(opts)
}

// createCore 创建日志核心，定义日志的编码器和写入器。
func createCore(encoder zapcore.Encoder, writer zapcore.WriteSyncer, level zapcore.Level) zapcore.Core {
	return zapcore.NewCore(encoder, writer, level)
}

// buildCores 构建日志核心，依据配置生成一个或多个日志输出核心。
func buildCores(opts Options, encodingConfig *zapcore.EncoderConfig) ([]zapcore.Core, error) {
	var cores []zapcore.Core

	// 使用 JSON 编码器输出到文件
	jsonEncoder := zapcore.NewJSONEncoder(*encodingConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(*encodingConfig)

	// 如果设置了 Filename，输出到文件
	if opts.Filename != "" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   opts.Filename,
			MaxSize:    opts.Rotation.MaxSize,
			MaxBackups: opts.Rotation.MaxBackups,
			MaxAge:     opts.Rotation.MaxAge,
			Compress:   opts.Rotation.Compress,
			LocalTime:  opts.Rotation.LocalTime,
		})
		cores = append(cores, createCore(jsonEncoder, fileWriter, zapcore.DebugLevel))
	}

	// 如果需要输出到控制台，使用 console 编码器
	if opts.Stdout {
		stdoutWriter := zapcore.AddSync(os.Stdout)
		cores = append(cores, createCore(consoleEncoder, stdoutWriter, opts.Level))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no output configured")
	}

	return cores, nil
}

// Write 实现 io.Writer 接口，用于标准库 log 的输出。
func (l *logger) Write(p []byte) (n int, err error) {
	l.Info(string(p))
	return len(p), nil
}

// processDebugFields 处理 Debug 模式下的字段，添加额外的调试信息。
func (l *logger) processDebugFields(fields []zapcore.Field) []zapcore.Field {
	return append(fields,
		zap.Stack("stack"),
		zap.Namespace("details"),
		zap.Time("debug_time", time.Now()),
	)
}

// Debug 记录调试级别的日志消息。
func (l *logger) Debug(msg string, fields ...zapcore.Field) {
	if l.isDebug {
		fields = l.processDebugFields(fields)
	}
	l.zap.Debug(msg, fields...)
}

// Info 记录信息级别的日志消息。
func (l *logger) Info(msg string, fields ...zapcore.Field)   { l.zap.Info(msg, fields...) }

// Warn 记录警告级别的日志消息。
func (l *logger) Warn(msg string, fields ...zapcore.Field)   { l.zap.Warn(msg, fields...) }

// Error 记录错误级别的日志消息。
func (l *logger) Error(msg string, fields ...zapcore.Field)  { l.zap.Error(msg, fields...) }

// DPanic 记录严重错误级别的日志消息，并在开发模式下引发 panic。
func (l *logger) DPanic(msg string, fields ...zapcore.Field) { l.zap.DPanic(msg, fields...) }

// Panic 记录 panic 级别的日志消息。
func (l *logger) Panic(msg string, fields ...zapcore.Field)  { l.zap.Panic(msg, fields...) }

// Fatal 记录致命级别的日志消息，并终止程序。
func (l *logger) Fatal(msg string, fields ...zapcore.Field)  { l.zap.Fatal(msg, fields...) }

// Debugf 使用格式化字符串记录调试级别的日志消息。
func (l *logger) Debugf(template string, args ...interface{})  { l.sugar.Debugf(template, args...) }

// Infof 使用格式化字符串记录信息级别的日志消息。
func (l *logger) Infof(template string, args ...interface{})   { l.sugar.Infof(template, args...) }

// Warnf 使用格式化字符串记录警告级别的日志消息。
func (l *logger) Warnf(template string, args ...interface{})   { l.sugar.Warnf(template, args...) }

// Errorf 使用格式化字符串记录错误级别的日志消息。
func (l *logger) Errorf(template string, args ...interface{})  { l.sugar.Errorf(template, args...) }

// DPanicf 使用格式化字符串记录严重错误级别的日志消息，并在开发模式下引发 panic。
func (l *logger) DPanicf(template string, args ...interface{}) { l.sugar.DPanicf(template, args...) }

// Panicf 使用格式化字符串记录 panic 级别的日志消息。
func (l *logger) Panicf(template string, args ...interface{})  { l.sugar.Panicf(template, args...) }

// Fatalf 使用格式化字符串记录致命级别的日志消息，并终止程序。
func (l *logger) Fatalf(template string, args ...interface{})  { l.sugar.Fatalf(template, args...) }

// Print 实现标准库 log 接口，输出日志消息。
func (l *logger) Print(v ...interface{})                 { l.stdLog.Print(v...) }

// Printf 实现标准库 log 接口，使用格式化字符串输出日志消息。
func (l *logger) Printf(format string, v ...interface{}) { l.stdLog.Printf(format, v...) }

// Println 实现标准库 log 接口，输出日志消息，并追加换行符。
func (l *logger) Println(v ...interface{})               { l.stdLog.Println(v...) }

// Sync 同步日志缓冲区，确保所有日志都被写入。
func (l *logger) Sync() error { return l.zap.Sync() }

// With 返回一个新的 Logger 实例，添加指定的字段。
func (l *logger) With(fields ...zapcore.Field) Logger {
	newLogger := &logger{
		zap:     l.zap.With(fields...),
		opts:    l.opts,
		isDebug: l.isDebug,
	}
	newLogger.sugar = newLogger.zap.Sugar()
	writer := zapcore.AddSync(newLogger)
	newLogger.stdLog = log.New(writer, "", 0)
	return newLogger
}
