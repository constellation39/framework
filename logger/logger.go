package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 封装了 zap.Logger 和相关资源
type Logger struct {
	*zap.Logger
	sugar      *zap.SugaredLogger
	rotateLog  io.Closer
	config     *Config
	callerOnce sync.Once
	callerPath string
}

// Config 日志配置
type Config struct {
	// 基础配置
	Level       string `json:"level" yaml:"level"`             // 日志级别: debug, info, warn, error
	Encoding    string `json:"encoding" yaml:"encoding"`       // 编码格式: json, console
	Environment string `json:"environment" yaml:"environment"` // 环境: development, production

	// 文件配置
	EnableFile     bool   `json:"enable_file" yaml:"enable_file"`           // 是否启用文件输出
	LogDir         string `json:"log_dir" yaml:"log_dir"`                   // 日志目录
	Filename       string `json:"filename" yaml:"filename"`                 // 日志文件名前缀
	MaxAge         int    `json:"max_age" yaml:"max_age"`                   // 日志保留天数
	RotationTime   int    `json:"rotation_time" yaml:"rotation_time"`       // 轮转时间(小时)
	RotationSize   int64  `json:"rotation_size" yaml:"rotation_size"`       // 轮转大小(MB)
	RotationCount  uint   `json:"rotation_count" yaml:"rotation_count"`     // 保留文件数量
	CompressOldLog bool   `json:"compress_old_log" yaml:"compress_old_log"` // 是否压缩旧日志

	// 控制台配置
	EnableConsole bool `json:"enable_console" yaml:"enable_console"` // 是否启用控制台输出
	ColorConsole  bool `json:"color_console" yaml:"color_console"`   // 控制台是否彩色输出

	// 高级配置
	EnableStacktrace bool   `json:"enable_stacktrace" yaml:"enable_stacktrace"` // 是否启用堆栈跟踪
	StacktraceLevel  string `json:"stacktrace_level" yaml:"stacktrace_level"`   // 堆栈跟踪级别
	MaxStackFrames   int    `json:"max_stack_frames" yaml:"max_stack_frames"`   // 最大堆栈帧数
	CallerSkip       int    `json:"caller_skip" yaml:"caller_skip"`             // 调用者跳过层数
	EnableSampling   bool   `json:"enable_sampling" yaml:"enable_sampling"`     // 是否启用采样
	SamplingInitial  int    `json:"sampling_initial" yaml:"sampling_initial"`   // 采样初始值
	SamplingAfter    int    `json:"sampling_after" yaml:"sampling_after"`       // 采样之后值
}

// 默认配置
func defaultConfig() *Config {
	return &Config{
		Level:            "info",
		Encoding:         "console",
		Environment:      "development",
		EnableFile:       true,
		LogDir:           "logs",
		Filename:         "app",
		MaxAge:           7,
		RotationTime:     24,
		RotationSize:     100,
		RotationCount:    10,
		CompressOldLog:   false,
		EnableConsole:    true,
		ColorConsole:     true,
		EnableStacktrace: true,
		StacktraceLevel:  "error",
		MaxStackFrames:   10,
		CallerSkip:       0,
		EnableSampling:   false,
		SamplingInitial:  100,
		SamplingAfter:    100,
	}
}

// New 创建新的日志实例
func New(opts ...Option) (*Logger, error) {
	cfg := defaultConfig()

	// 应用选项
	for _, opt := range opts {
		opt(cfg)
	}

	// 根据环境自动调整默认值
	if cfg.Environment == "production" {
		if cfg.Encoding == "console" {
			cfg.Encoding = "json"
		}
		if !cfg.EnableSampling {
			cfg.EnableSampling = true
		}
		cfg.ColorConsole = false
	}

	return newLogger(cfg)
}

// MustNew 创建日志实例，失败则 panic
func MustNew(opts ...Option) *Logger {
	logger, err := New(opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	return logger
}

func newLogger(cfg *Config) (*Logger, error) {
	// 解析日志级别
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}

	// 解析堆栈跟踪级别
	stackLevel, err := parseLevel(cfg.StacktraceLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid stacktrace level %s: %w", cfg.StacktraceLevel, err)
	}

	// 构建 cores
	cores := make([]zapcore.Core, 0, 2)
	var rotateLog io.Closer

	// 文件输出
	if cfg.EnableFile {
		fileCore, rl, err := buildFileCore(cfg, level)
		if err != nil {
			return nil, fmt.Errorf("failed to build file core: %w", err)
		}
		cores = append(cores, fileCore)
		rotateLog = rl
	}

	// 控制台输出
	if cfg.EnableConsole {
		consoleCore := buildConsoleCore(cfg, level)
		cores = append(cores, consoleCore)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("at least one output (file or console) must be enabled")
	}

	// 组合多个 core
	core := zapcore.NewTee(cores...)

	// 包装堆栈截断
	if cfg.EnableStacktrace && cfg.MaxStackFrames > 0 {
		core = &stackTrimCore{
			Core:      core,
			maxFrames: cfg.MaxStackFrames,
		}
	}

	// 构建选项
	zapOpts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(cfg.CallerSkip),
	}

	if cfg.EnableStacktrace {
		zapOpts = append(zapOpts, zap.AddStacktrace(stackLevel))
	}

	// 添加采样
	if cfg.EnableSampling {
		zapOpts = append(zapOpts, zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				c,
				time.Second,
				cfg.SamplingInitial,
				cfg.SamplingAfter,
			)
		}))
	}

	// 创建 logger
	zapLogger := zap.New(core, zapOpts...)

	logger := &Logger{
		Logger:    zapLogger,
		sugar:     zapLogger.Sugar(),
		rotateLog: rotateLog,
		config:    cfg,
	}

	return logger, nil
}

// buildFileCore 构建文件输出 core
func buildFileCore(cfg *Config, level zapcore.Level) (zapcore.Core, io.Closer, error) {
	// 创建日志目录
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 构建日志文件路径
	logPath := filepath.Join(cfg.LogDir, cfg.Filename+".%Y%m%d.log")
	linkPath := filepath.Join(cfg.LogDir, cfg.Filename+".log")

	// 配置 rotatelogs
	// Note: the rotatelogs library does not allow MaxAge and RotationCount
	// to be set simultaneously. Prefer RotationCount when provided by the
	// user/config; otherwise fall back to MaxAge.
	rotateOpts := []rotatelogs.Option{
		rotatelogs.WithLinkName(linkPath),
		rotatelogs.WithRotationTime(time.Duration(cfg.RotationTime) * time.Hour),
	}

	if cfg.RotationSize > 0 {
		rotateOpts = append(rotateOpts, rotatelogs.WithRotationSize(cfg.RotationSize*1024*1024))
	}

	if cfg.RotationCount > 0 {
		// Use RotationCount and do not set MaxAge
		rotateOpts = append(rotateOpts, rotatelogs.WithRotationCount(cfg.RotationCount))
	} else if cfg.MaxAge > 0 {
		rotateOpts = append(rotateOpts, rotatelogs.WithMaxAge(time.Duration(cfg.MaxAge) * 24 * time.Hour))
	}

	// 创建 rotatelogs
	logWriter, err := rotatelogs.New(logPath, rotateOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rotatelogs: %w", err)
	}

	// 构建编码器
	encoder := buildEncoder(cfg, false)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(logWriter),
		level,
	)

	return core, logWriter, nil
}

// buildConsoleCore 构建控制台输出 core
func buildConsoleCore(cfg *Config, level zapcore.Level) zapcore.Core {
	encoder := buildEncoder(cfg, true)

	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
}

// buildEncoder 构建编码器
func buildEncoder(cfg *Config, isConsole bool) zapcore.Encoder {
	var encoderConfig zapcore.EncoderConfig

	if cfg.Environment == "production" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 统一字段名
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.NameKey = "logger"
	encoderConfig.CallerKey = "caller"
	encoderConfig.FunctionKey = zapcore.OmitKey
	encoderConfig.MessageKey = "msg"
	encoderConfig.StacktraceKey = "stacktrace"
	encoderConfig.LineEnding = zapcore.DefaultLineEnding
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.MillisDurationEncoder

	// 控制台特殊配置
	if isConsole {
		encoderConfig.EncodeCaller = relativeCallerEncoder
		if cfg.ColorConsole {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	} else {
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	}

	// 根据编码格式创建编码器
	if cfg.Encoding == "json" || !isConsole {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// parseLevel 解析日志级别
func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn", "warning":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	case "dpanic":
		return zap.DPanicLevel, nil
	case "panic":
		return zap.PanicLevel, nil
	case "fatal":
		return zap.FatalLevel, nil
	default:
		return zap.InfoLevel, fmt.Errorf("unknown level: %s", level)
	}
}

// relativeCallerEncoder 相对路径编码器
func relativeCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// 只计算一次工作目录
	l := globalLogger.Load()
	if l == nil {
		zapcore.ShortCallerEncoder(caller, enc)
		return
	}

	logger := l.(*Logger)
	logger.callerOnce.Do(func() {
		logger.callerPath, _ = os.Getwd()
	})

	if logger.callerPath == "" {
		zapcore.ShortCallerEncoder(caller, enc)
		return
	}

	relPath, err := filepath.Rel(logger.callerPath, caller.File)
	if err != nil {
		zapcore.ShortCallerEncoder(caller, enc)
		return
	}

	enc.AppendString(fmt.Sprintf("%s:%d", relPath, caller.Line))
}

// stackTrimCore 堆栈截断核心
type stackTrimCore struct {
	zapcore.Core
	maxFrames int
}

func (c *stackTrimCore) With(fields []zapcore.Field) zapcore.Core {
	return &stackTrimCore{
		Core:      c.Core.With(fields),
		maxFrames: c.maxFrames,
	}
}

func (c *stackTrimCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *stackTrimCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// 复制 Entry 避免修改原始数据
	newEnt := ent
	if newEnt.Stack != "" {
		lines := strings.Split(newEnt.Stack, "\n")
		maxLines := c.maxFrames * 2 // 每个帧包含函数名和文件位置两行
		if len(lines) > maxLines {
			newEnt.Stack = strings.Join(lines[:maxLines], "\n") + "\n\t... stack trimmed ..."
		}
	}
	return c.Core.Write(newEnt, fields)
}

// Sugar 返回 SugaredLogger
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

// Sync 刷新缓冲区
func (l *Logger) Sync() error {
	if err := l.Logger.Sync(); err != nil {
		// 忽略 stdout/stderr 的 sync 错误
		if !strings.Contains(err.Error(), "inappropriate ioctl for device") {
			return err
		}
	}
	return nil
}

// Close 关闭日志
func (l *Logger) Close() error {
	// 先刷新缓冲区
	if err := l.Sync(); err != nil {
		return err
	}

	// 关闭 rotatelogs
	if l.rotateLog != nil {
		return l.rotateLog.Close()
	}

	return nil
}

// GetConfig 获取配置
func (l *Logger) GetConfig() *Config {
	cfg := *l.config
	return &cfg
}
