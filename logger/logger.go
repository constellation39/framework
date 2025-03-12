package logger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v2"
)

/*
Package logger 提供了一个基于zap的日志系统，具有以下特性：
- 简单易用的API
- 高性能的日志记录
- 灵活的配置选项
- 上下文感知
- 文件轮转支持
- 多级别输出控制
- 配置文件支持
- 错误恢复机制

基本使用:
    logger, _ := logger.New(logger.WithLevel(zapcore.InfoLevel))
    logger.Info("这是一条信息日志")
    logger.Error("发生错误", zap.Error(err))

上下文使用:
    ctx := context.Background()
    ctxLogger, _ := logger.NewLoggerWithContext(ctx)
    ctxLogger.Info("带有上下文的日志")

配置文件使用:
    logger, _ := logger.FromConfig("config.yaml")
*/

// 默认值常量
const (
	DefaultMaxSize    = 100 // 默认日志文件最大尺寸(MB)
	DefaultMaxBackups = 3   // 默认保留的备份数
	DefaultMaxAge     = 28  // 默认日志保留天数
)

// 预定义错误
var (
	ErrNoOutputConfigured = errors.New("logger: no output configured")
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
			MaxSize:    DefaultMaxSize,
			MaxBackups: DefaultMaxBackups,
			MaxAge:     DefaultMaxAge,
			Compress:   true,
			LocalTime:  true,
		},
		CallerSkip: 0,
	}
}

// LoggerOption 定义一个修改Logger选项的函数类型
type LoggerOption func(*Options)

// WithLevel 设置日志级别
func WithLevel(level zapcore.Level) LoggerOption {
	return func(o *Options) {
		o.Level = level
	}
}

// WithStdout 设置是否输出到控制台
func WithStdout(enable bool) LoggerOption {
	return func(o *Options) {
		o.Stdout = enable
	}
}

// WithFile 设置日志文件
func WithFile(filename string) LoggerOption {
	return func(o *Options) {
		o.Filename = filename
	}
}

// WithRotation 设置日志轮转选项
func WithRotation(opts RotationOptions) LoggerOption {
	return func(o *Options) {
		o.Rotation = opts
	}
}

// WithRotationValues 通过直接值设置日志轮转选项
func WithRotationValues(maxSize, maxBackups, maxAge int, compress, localTime bool) LoggerOption {
	return WithRotation(RotationOptions{
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
		LocalTime:  localTime,
	})
}

// WithCallerSkip 设置调用者跳过层级
func WithCallerSkip(skip int) LoggerOption {
	return func(o *Options) {
		o.CallerSkip = skip
	}
}

// WithFields 添加默认字段
func WithFields(fields ...zap.Field) LoggerOption {
	return func(o *Options) {
		o.Fields = append(o.Fields, fields...)
	}
}

// newEncoderConfig 返回标准的编码器配置
func newEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
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
}

// New 使用函数选项模式创建日志器
func New(opts ...LoggerOption) (*zap.Logger, error) {
	// 应用默认选项
	options := DefaultOptions()

	// 应用用户提供的选项
	for _, opt := range opts {
		opt(&options)
	}

	var cores []zapcore.Core

	// 添加文件输出
	if options.Filename != "" {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   options.Filename,
			MaxSize:    options.Rotation.MaxSize,
			MaxBackups: options.Rotation.MaxBackups,
			MaxAge:     options.Rotation.MaxAge,
			Compress:   options.Rotation.Compress,
			LocalTime:  options.Rotation.LocalTime,
		}

		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(newEncoderConfig()),
			zapcore.AddSync(lumberjackLogger),
			options.Level,
		)
		cores = append(cores, fileCore)
	}

	// 添加控制台输出
	if options.Stdout {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(newEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			options.Level,
		)
		cores = append(cores, consoleCore)
	}

	// 检查是否有输出配置
	if len(cores) == 0 {
		return nil, ErrNoOutputConfigured
	}

	// 构建基本 Zap 选项
	zapOpts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.WarnLevel),
	}

	// 添加额外选项
	if options.Level == zapcore.DebugLevel {
		zapOpts = append(zapOpts, zap.Development())
	}

	if options.CallerSkip > 0 {
		zapOpts = append(zapOpts, zap.AddCallerSkip(options.CallerSkip))
	}

	if len(options.Fields) > 0 {
		zapOpts = append(zapOpts, zap.Fields(options.Fields...))
	}

	// 创建并返回 logger
	return zap.New(zapcore.NewTee(cores...), zapOpts...), nil
}

// MustNew 创建日志器或panic
func MustNew(opts ...LoggerOption) *zap.Logger {
	logger, err := New(opts...)
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %w", err))
	}
	return logger
}

// NewDefaultLogger 创建默认日志器
func NewDefaultLogger() (*zap.Logger, error) {
	return New()
}

// MustNewDefaultLogger 创建默认日志器或panic
func MustNewDefaultLogger() *zap.Logger {
	return MustNew()
}

// NewDevelopmentLogger 创建适合开发环境的日志器
func NewDevelopmentLogger() (*zap.Logger, error) {
	return New(WithLevel(zapcore.DebugLevel))
}

// NewProductionLogger 创建适合生产环境的日志器
func NewProductionLogger(filename string) (*zap.Logger, error) {
	if filename == "" {
		return nil, fmt.Errorf("production logger requires a filename")
	}
	return New(
		WithLevel(zapcore.InfoLevel),
		WithStdout(false),
		WithFile(filename),
	)
}

// extractFieldsFromContext 从上下文中提取字段
func extractFieldsFromContext(ctx context.Context) []zap.Field {
	var fields []zap.Field
	// 这里可以根据需要从上下文提取信息转换为字段
	return fields
}

// NewLoggerWithContext 创建带有上下文信息的日志器
func NewLoggerWithContext(ctx context.Context, options ...LoggerOption) (*zap.Logger, error) {
	// 从上下文中提取字段
	fields := extractFieldsFromContext(ctx)

	// 将上下文字段添加到选项中
	ctxOptions := append([]LoggerOption{WithFields(fields...)}, options...)

	// 创建日志器并直接返回zap.Logger
	return New(ctxOptions...)
}

// WithContext 为现有的logger添加上下文信息
func WithContext(logger *zap.Logger, ctx context.Context) *zap.Logger {
	if logger == nil || ctx == nil {
		return logger
	}

	fields := extractFieldsFromContext(ctx)
	if len(fields) == 0 {
		return logger
	}

	return logger.With(fields...)
}

// 配置文件支持

// ConfigFileFormat 表示配置文件格式
type ConfigFileFormat int

const (
	FormatJSON ConfigFileFormat = iota
	FormatYAML
)

// FromConfig 从配置文件创建日志器
func FromConfig(configPath string) (*zap.Logger, error) {
	format := FormatJSON
	if strings.HasSuffix(strings.ToLower(configPath), ".yaml") ||
		strings.HasSuffix(strings.ToLower(configPath), ".yml") {
		format = FormatYAML
	}

	config, err := loadConfigFile(configPath, format)
	if err != nil {
		return nil, fmt.Errorf("failed to load logger config: %w", err)
	}

	// 转换配置到选项
	opts := []LoggerOption{
		WithLevel(config.Level),
		WithStdout(config.Stdout),
		WithCallerSkip(config.CallerSkip),
	}

	if config.Filename != "" {
		opts = append(opts, WithFile(config.Filename))
		opts = append(opts, WithRotation(config.Rotation))
	}

	if len(config.Fields) > 0 {
		opts = append(opts, WithFields(config.Fields...))
	}

	// 创建日志器
	return New(opts...)
}

// loadConfigFile 从文件加载配置
func loadConfigFile(path string, format ConfigFileFormat) (Options, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return DefaultOptions(), fmt.Errorf("failed to read config file: %w", err)
	}

	options := DefaultOptions()

	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &options); err != nil {
			return DefaultOptions(), fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(data, &options); err != nil {
			return DefaultOptions(), fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return DefaultOptions(), fmt.Errorf("unsupported config format")
	}

	return options, nil
}

// MustFromConfig 从配置文件创建日志器或panic
func MustFromConfig(configPath string) *zap.Logger {
	logger, err := FromConfig(configPath)
	if err != nil {
		panic(fmt.Errorf("failed to create logger from config: %w", err))
	}
	return logger
}

// 全局日志器和辅助函数

// 全局日志器，可以通过SetGlobalLogger设置
var globalLogger *zap.Logger = zap.NewNop()

// SetGlobalLogger 设置全局日志器
func SetGlobalLogger(logger *zap.Logger) {
	if logger != nil {
		globalLogger = logger
	}
}

// GetGlobalLogger 获取全局日志器
func GetGlobalLogger() *zap.Logger {
	return globalLogger
}

// 快速日志记录函数

// Debug 使用全局日志器记录调试信息
func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

// Info 使用全局日志器记录一般信息
func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

// Warn 使用全局日志器记录警告信息
func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

// Error 使用全局日志器记录错误信息
func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

// DPanic 使用全局日志器记录严重错误
func DPanic(msg string, fields ...zap.Field) {
	globalLogger.DPanic(msg, fields...)
}

// Panic 使用全局日志器记录并触发panic
func Panic(msg string, fields ...zap.Field) {
	globalLogger.Panic(msg, fields...)
}

// Fatal 使用全局日志器记录致命错误并退出
func Fatal(msg string, fields ...zap.Field) {
	globalLogger.Fatal(msg, fields...)
}

// WithError 快速创建带错误字段的日志
func WithError(logger *zap.Logger, err error) *zap.Logger {
	if err == nil {
		return logger
	}
	return logger.With(zap.Error(err))
}

// Sync 同步所有日志缓冲区到输出
func Sync() error {
	return globalLogger.Sync()
}
