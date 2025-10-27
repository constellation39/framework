package logger

import (
	"os"
	"strings"
)

// Option 配置选项
type Option func(*Config)

// WithLevel 设置日志级别
func WithLevel(level string) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithEncoding 设置编码格式
func WithEncoding(encoding string) Option {
	return func(c *Config) {
		c.Encoding = encoding
	}
}

// WithEnvironment 设置环境
func WithEnvironment(env string) Option {
	return func(c *Config) {
		c.Environment = env
	}
}

// WithProductionMode 设置为生产模式
func WithProductionMode() Option {
	return func(c *Config) {
		c.Environment = "production"
	}
}

// WithDevelopmentMode 设置为开发模式
func WithDevelopmentMode() Option {
	return func(c *Config) {
		c.Environment = "development"
	}
}

// WithFile 配置文件输出
func WithFile(enabled bool, dir, filename string) Option {
	return func(c *Config) {
		c.EnableFile = enabled
		if dir != "" {
			c.LogDir = dir
		}
		if filename != "" {
			c.Filename = filename
		}
	}
}

// WithRotation 配置日志轮转
func WithRotation(maxAge, rotationTime int, rotationSize int64) Option {
	return func(c *Config) {
		if maxAge > 0 {
			c.MaxAge = maxAge
		}
		if rotationTime > 0 {
			c.RotationTime = rotationTime
		}
		if rotationSize > 0 {
			c.RotationSize = rotationSize
		}
	}
}

// WithRotationCount 设置保留文件数量
func WithRotationCount(count uint) Option {
	return func(c *Config) {
		c.RotationCount = count
	}
}

// WithCompression 启用日志压缩
func WithCompression(enabled bool) Option {
	return func(c *Config) {
		c.CompressOldLog = enabled
	}
}

// WithConsole 配置控制台输出
func WithConsole(enabled bool, colored bool) Option {
	return func(c *Config) {
		c.EnableConsole = enabled
		c.ColorConsole = colored
	}
}

// WithStacktrace 配置堆栈跟踪
func WithStacktrace(enabled bool, level string, maxFrames int) Option {
	return func(c *Config) {
		c.EnableStacktrace = enabled
		if level != "" {
			c.StacktraceLevel = level
		}
		if maxFrames > 0 {
			c.MaxStackFrames = maxFrames
		}
	}
}

// WithCallerSkip 设置调用者跳过层数
func WithCallerSkip(skip int) Option {
	return func(c *Config) {
		c.CallerSkip = skip
	}
}

// WithSampling 配置采样
func WithSampling(enabled bool, initial, after int) Option {
	return func(c *Config) {
		c.EnableSampling = enabled
		if initial > 0 {
			c.SamplingInitial = initial
		}
		if after > 0 {
			c.SamplingAfter = after
		}
	}
}

// WithConfig 使用完整配置
func WithConfig(cfg *Config) Option {
	return func(c *Config) {
		*c = *cfg
	}
}

// WithEnvAutoConfig 根据环境变量自动配置
func WithEnvAutoConfig() Option {
	return func(c *Config) {
		// LOG_LEVEL
		if level := os.Getenv("LOG_LEVEL"); level != "" {
			c.Level = level
		}

		// LOG_ENV or ENV
		if env := os.Getenv("LOG_ENV"); env != "" {
			c.Environment = env
		} else if env := os.Getenv("ENV"); env != "" {
			c.Environment = env
		}

		// 根据环境自动调整
		if strings.EqualFold(c.Environment, "production") ||
			strings.EqualFold(c.Environment, "prod") {
			c.Environment = "production"
			c.Encoding = "json"
			c.EnableSampling = true
			c.ColorConsole = false
		}

		// LOG_DIR
		if dir := os.Getenv("LOG_DIR"); dir != "" {
			c.LogDir = dir
		}

		// LOG_FILE_ENABLED
		if enabled := os.Getenv("LOG_FILE_ENABLED"); enabled != "" {
			c.EnableFile = strings.EqualFold(enabled, "true")
		}

		// LOG_CONSOLE_ENABLED
		if enabled := os.Getenv("LOG_CONSOLE_ENABLED"); enabled != "" {
			c.EnableConsole = strings.EqualFold(enabled, "true")
		}
	}
}
