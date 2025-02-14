package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 定义日志接口，包含常用日志方法和本例中的新方法 WithCallerSkip。
type Logger interface {
	Debug(msg string, fields ...zapcore.Field)
	Info(msg string, fields ...zapcore.Field)
	Warn(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	DPanic(msg string, fields ...zapcore.Field)
	Panic(msg string, fields ...zapcore.Field)
	Fatal(msg string, fields ...zapcore.Field)

	// Sync 用于刷新缓冲区，一般在程序退出前调用。
	Sync() error

	// WithCallerSkip 创建一个新的 Logger 并调整 caller skip。
	WithCallerSkip(skip int) Logger

	// With 也是常用功能：生成新的 Logger 并携带一些固定字段。
	With(fields ...zapcore.Field) Logger
}

// Options 定义日志配置选项
type Options struct {
	// Level 指定日志级别，比如 zapcore.DebugLevel, zapcore.InfoLevel 等
	Level zapcore.Level

	// Filename 指定日志输出文件名，不为空则文件输出，空则只输出到控制台
	Filename string

	// Stdout 为 true 时会输出到控制台
	Stdout bool

	// 文件轮转配置
	Rotation RotationOptions

	// CallerSkip 设置 zap 内部调用层数的跳过数量
	CallerSkip int

	// Fields 允许在 logger 初始化时就预带特定字段
	Fields []zap.Field
}

// RotationOptions 定义日志轮转配置
type RotationOptions struct {
	MaxSize    int  // MB
	MaxBackups int  // 保留旧文件个数
	MaxAge     int  // 保留天数
	Compress   bool // 是否压缩
	LocalTime  bool // 使用本地时间
}

// DefaultOptions 返回一套默认日志配置
func DefaultOptions() Options {
	return Options{
		Level:    zapcore.InfoLevel,
		Stdout:   true,
		Filename: "",
		Rotation: RotationOptions{
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
			LocalTime:  true,
		},
		// 通常在 logger 包内封装一层时设为 1
		CallerSkip: 1,
	}
}

// zapLogger 是 Logger 接口的具体实现，基于 zap。
type zapLogger struct {
	base *zap.Logger
	// 保存 Options，以便二次复制或更新
	opts Options
	// 可选：你也可以保留 sugared logger 或 stdLog 等
}

// New 创建新的 Logger (基于 zap) 并返回 Logger 接口
func New(opts Options) (Logger, error) {
	// 构建底层 zap.Logger
	zLogger, err := newZapLogger(opts)
	if err != nil {
		return nil, err
	}
	return &zapLogger{
		base: zLogger,
		opts: opts,
	}, nil
}

// 下面是辅助函数 newZapLogger 用于构建 raw zap.Logger
func newZapLogger(opts Options) (*zap.Logger, error) {
	encodingConfig := zapcore.EncoderConfig{
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

	var cores []zapcore.Core

	// 如果指定了文件名，则输出到文件(使用 JSON 编码)
	if opts.Filename != "" {
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   opts.Filename,
			MaxSize:    opts.Rotation.MaxSize,
			MaxBackups: opts.Rotation.MaxBackups,
			MaxAge:     opts.Rotation.MaxAge,
			Compress:   opts.Rotation.Compress,
			LocalTime:  opts.Rotation.LocalTime,
		})
		jsonEncoder := zapcore.NewJSONEncoder(encodingConfig)
		cores = append(cores, zapcore.NewCore(jsonEncoder, fileWriter, zapcore.DebugLevel))
	}

	// 若需要输出到控制台，则使用 ConsoleEncoder
	if opts.Stdout {
		consoleEncoder := zapcore.NewConsoleEncoder(encodingConfig)
		consoleWriter := zapcore.AddSync(os.Stdout)
		cores = append(cores, zapcore.NewCore(consoleEncoder, consoleWriter, opts.Level))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no output configured")
	}

	zapOpts := []zap.Option{
		zap.AddCaller(),
	}
	// 如果是 Debug 级别，可加入 zap.Development() 方便调试
	if opts.Level == zapcore.DebugLevel {
		zapOpts = append(zapOpts, zap.Development())
	}
	// 如果设置了 callerSkip
	if opts.CallerSkip > 0 {
		zapOpts = append(zapOpts, zap.AddCallerSkip(opts.CallerSkip))
	}
	// 如果有初始需要附加的字段
	if len(opts.Fields) > 0 {
		zapOpts = append(zapOpts, zap.Fields(opts.Fields...))
	}

	// 构建 zap.Logger
	return zap.New(zapcore.NewTee(cores...),
		zapOpts...,
	), nil
}

// -------------------------------------------
// 实现 Logger 接口的方法
// -------------------------------------------

func (l *zapLogger) Debug(msg string, fields ...zapcore.Field) {
	l.base.Debug(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...zapcore.Field) {
	l.base.Info(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...zapcore.Field) {
	l.base.Warn(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...zapcore.Field) {
	l.base.Error(msg, fields...)
}

func (l *zapLogger) DPanic(msg string, fields ...zapcore.Field) {
	l.base.DPanic(msg, fields...)
}

func (l *zapLogger) Panic(msg string, fields ...zapcore.Field) {
	l.base.Panic(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...zapcore.Field) {
	l.base.Fatal(msg, fields...)
}

// Sync 用于刷新写缓冲
func (l *zapLogger) Sync() error {
	return l.base.Sync()
}

// With 用于在当前 Logger 基础上附加一些字段
func (l *zapLogger) With(fields ...zapcore.Field) Logger {
	newLogger := &zapLogger{
		// 直接复用当前 zap.Logger，然后调用 With(fields...)
		base: l.base.With(fields...),
		opts: l.opts,
	}
	return newLogger
}

// WithCallerSkip 根据当前 logger 的配置，创建一个新的 logger
// 并增加(或覆盖) callerSkip，使日志能正确定位到你想要的调用层级。
func (l *zapLogger) WithCallerSkip(skip int) Logger {
	// 复制一份原有 opts
	newOpts := l.opts
	// 覆盖 callerSkip
	newOpts.CallerSkip = skip

	// 可选：这里也能动态添加“module”字段，便于区分日志来源。
	// 下面是示例：用 runtime 获取调用方函数名
	if len(newOpts.Fields) == 0 {
		newOpts.Fields = make([]zap.Field, 0)
	}
	if pc, _, _, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			newOpts.Fields = append(newOpts.Fields, zap.String("module", path.Base(fn.Name())))
		}
	}

	zLogger, err := newZapLogger(newOpts)
	if err != nil {
		// 如果因为某些不常见错误创建失败，这里可以返回原 logger 以防止 crash
		return l
	}

	return &zapLogger{
		base: zLogger,
		opts: newOpts,
	}
}

// NewDefaultLogger 快速创建一个默认 logger
func NewDefaultLogger() Logger {
	l, err := New(DefaultOptions())
	if err != nil {
		panic(fmt.Sprintf("failed to create default logger: %v", err))
	}
	return l
}
