package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

// Options 定义日志配置选项
type Options struct {
	Level      zapcore.Level // 日志级别
	Filename   string        // 文件名，不为空则输出到文件
	Stdout     bool          // 是否同时输出到控制台
	Rotation   RotationOptions
	CallerSkip int         // 在封装场景下需要的 caller skip
	Fields     []zap.Field // 初始化时默认附加的字段
}

// RotationOptions 定义日志轮转配置
type RotationOptions struct {
	MaxSize    int  // MB
	MaxBackups int  // 保留旧文件个数
	MaxAge     int  // 保留天数
	Compress   bool // 是否压缩
	LocalTime  bool // 是否使用本地时间
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
		CallerSkip: 0,
	}
}

// NewLogger 以最小封装的方式，返回 *zap.Logger
func NewLogger(opts Options) (*zap.Logger, error) {
	encodingConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
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

	// 如果 Filename 不为空，则输出到文件；此处用 JSON 编码
	if opts.Filename != "" {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   opts.Filename,
			MaxSize:    opts.Rotation.MaxSize,
			MaxBackups: opts.Rotation.MaxBackups,
			MaxAge:     opts.Rotation.MaxAge,
			Compress:   opts.Rotation.Compress,
			LocalTime:  opts.Rotation.LocalTime,
		})
		jsonEnc := zapcore.NewJSONEncoder(encodingConfig)
		// 设置为 DebugLevel 是为了让文件里面可以吃到比 console 更多的日志
		cores = append(cores, zapcore.NewCore(jsonEnc, w, zapcore.DebugLevel))
	}

	// 如果 Stdout 为 true，则再输出到控制台；此处用 Console 编码
	if opts.Stdout {
		consoleEnc := zapcore.NewConsoleEncoder(encodingConfig)
		consoleWriter := zapcore.AddSync(os.Stdout)
		cores = append(cores, zapcore.NewCore(consoleEnc, consoleWriter, opts.Level))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no output configured")
	}

	zapOpts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.WarnLevel),
	}
	// 如果要在 debug 模式下，需要 zap.Development()
	if opts.Level == zapcore.DebugLevel {
		zapOpts = append(zapOpts, zap.Development())
	}
	// 如果有 callerSkip
	if opts.CallerSkip > 0 {
		zapOpts = append(zapOpts, zap.AddCallerSkip(opts.CallerSkip))
	}
	// 如果有初始化 Fields
	if len(opts.Fields) > 0 {
		zapOpts = append(zapOpts, zap.Fields(opts.Fields...))
	}

	logger := zap.New(zapcore.NewTee(cores...), zapOpts...)
	return logger, nil
}

// NewDefaultLogger 快速创建一个默认 logger (若你想直接用)
func NewDefaultLogger() *zap.Logger {
	opts := DefaultOptions()
	logger, err := NewLogger(opts)
	if err != nil {
		panic(fmt.Errorf("failed to create default logger: %v", err))
	}
	return logger
}
