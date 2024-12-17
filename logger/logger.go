package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"time"
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

	// 编码配置
	Encoding       string // json 或 console
	EncodingConfig *zapcore.EncoderConfig

	// 扩展选项
	Development bool
	CallerSkip  int
	Fields      []zap.Field // 初始字段
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
		EncodingConfig: &zapcore.EncoderConfig{
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
		},
		Rotation: RotationOptions{
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
			LocalTime:  true,
		},
		Development: false,
		CallerSkip:  1,
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

// New 创建新的日志实例
func New(options Options) (Logger, error) {
	cores, err := buildCores(options)
	if err != nil {
		return nil, fmt.Errorf("build cores failed: %w", err)
	}

	zapOpts := []zap.Option{zap.AddCaller()}
	if options.Development {
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

func NewDefaultLogger() (Logger, error) {
	opts := DefaultOptions()
	return New(opts)
}

// buildCores 构建日志核心
func buildCores(opts Options) ([]zapcore.Core, error) {
	var cores []zapcore.Core

	jsonEncoder := zapcore.NewJSONEncoder(*opts.EncodingConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(*opts.EncodingConfig)

	if opts.Filename != "" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   opts.Filename,
			MaxSize:    opts.Rotation.MaxSize,
			MaxBackups: opts.Rotation.MaxBackups,
			MaxAge:     opts.Rotation.MaxAge,
			Compress:   opts.Rotation.Compress,
			LocalTime:  opts.Rotation.LocalTime,
		})
		cores = append(cores, zapcore.NewCore(jsonEncoder, fileWriter, zapcore.DebugLevel))
	}

	if opts.Stdout {
		stdoutWriter := zapcore.AddSync(os.Stdout)
		cores = append(cores, zapcore.NewCore(consoleEncoder, stdoutWriter, opts.Level))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no output configured")
	}

	return cores, nil
}

// Write 实现 io.Writer 接口，用于标准库 log 的输出
func (l *logger) Write(p []byte) (n int, err error) {
	l.Info(string(p))
	return len(p), nil
}

// 实现 Logger 接口的方法
func (l *logger) Debug(msg string, fields ...zapcore.Field) {
	if l.isDebug {
		// Debug模式下添加更多详细信息
		fields = append(fields,
			zap.Stack("stack"),
			zap.Namespace("details"),
			zap.Time("debug_time", time.Now()),
		)
	}
	l.zap.Debug(msg, fields...)
}

func (l *logger) Info(msg string, fields ...zapcore.Field)   { l.zap.Info(msg, fields...) }
func (l *logger) Warn(msg string, fields ...zapcore.Field)   { l.zap.Warn(msg, fields...) }
func (l *logger) Error(msg string, fields ...zapcore.Field)  { l.zap.Error(msg, fields...) }
func (l *logger) DPanic(msg string, fields ...zapcore.Field) { l.zap.DPanic(msg, fields...) }
func (l *logger) Panic(msg string, fields ...zapcore.Field)  { l.zap.Panic(msg, fields...) }
func (l *logger) Fatal(msg string, fields ...zapcore.Field)  { l.zap.Fatal(msg, fields...) }

func (l *logger) Debugf(template string, args ...interface{})  { l.sugar.Debugf(template, args...) }
func (l *logger) Infof(template string, args ...interface{})   { l.sugar.Infof(template, args...) }
func (l *logger) Warnf(template string, args ...interface{})   { l.sugar.Warnf(template, args...) }
func (l *logger) Errorf(template string, args ...interface{})  { l.sugar.Errorf(template, args...) }
func (l *logger) DPanicf(template string, args ...interface{}) { l.sugar.DPanicf(template, args...) }
func (l *logger) Panicf(template string, args ...interface{})  { l.sugar.Panicf(template, args...) }
func (l *logger) Fatalf(template string, args ...interface{})  { l.sugar.Fatalf(template, args...) }

// 实现标准库 log 接口
func (l *logger) Print(v ...interface{})                 { l.stdLog.Print(v...) }
func (l *logger) Printf(format string, v ...interface{}) { l.stdLog.Printf(format, v...) }
func (l *logger) Println(v ...interface{})               { l.stdLog.Println(v...) }

func (l *logger) Sync() error { return l.zap.Sync() }

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
