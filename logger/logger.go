package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

type Config struct {
	Level      zapcore.Level `json:"level"`      // 日志级别，有效值包括 "debug"、"info"、"warn"、"error" 和 "fatal"
	Filename   string        `json:"filename"`   // 日志文件的位置
	MaxSize    int           `json:"maxSize"`    // 单个日志文件的最大大小，单位为兆字节
	MaxBackups int           `json:"maxBackups"` // 保留的旧日志文件的最大数量
	MaxAge     int           `json:"maxAge"`     // 保留旧日志文件的最大天数
	Compress   bool          `json:"compress"`   // 是否压缩旧的日志文件
	Stdout     bool          `json:"stdout"`     // 是否将日志输出到标准输出（控制台）
}

var log *zap.Logger

const ErrNotInit = `logger not initialized, please call logger.Init(config) first.`

func Init(config Config) {
	var fileWriters []zapcore.WriteSyncer
	var consoleWriters []zapcore.WriteSyncer

	fileW := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize, // megabytes
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge, // days
		Compress:   config.Compress,
	})
	fileWriters = append(fileWriters, fileW)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderCfg.EncodeDuration = zapcore.StringDurationEncoder
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	var cores []zapcore.Core
	if config.Stdout {
		consoleWriters = append(consoleWriters, zapcore.AddSync(os.Stdout))

		consoleLevel := zapcore.InfoLevel
		if config.Level >= consoleLevel {
			consoleLevel = config.Level
		}
		consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.NewMultiWriteSyncer(consoleWriters...), consoleLevel)
		cores = append(cores, consoleCore)
	}

	fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.NewMultiWriteSyncer(fileWriters...), config.Level)
	cores = append(cores, fileCore)

	core := zapcore.NewTee(cores...)
	log = zap.New(core, zap.AddCaller(), zap.Development(), zap.AddCallerSkip(1))
}

func Info(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Info(msg, fields...)
}

func Warn(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Warn(msg, fields...)
}

func Error(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Error(msg, fields...)
}

func DPanic(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.DPanic(msg, fields...)
}

func Panic(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Fatal(msg, fields...)
}

func Debug(msg string, fields ...zapcore.Field) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Debug(msg, fields...)
}

func Infof(template string, args ...interface{}) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Sugar().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Sugar().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Sugar().Errorf(template, args...)
}

func DPanicf(template string, args ...interface{}) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Sugar().DPanicf(template, args...)
}

func Panicf(template string, args ...interface{}) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Sugar().Panicf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	if log == nil {
		panic(ErrNotInit)
	}
	log.Sugar().Fatalf(template, args...)
}

func Sync() error {
	if log == nil {
		panic(ErrNotInit)
	}
	return log.Sync()
}
