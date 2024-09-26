package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

type LogSync func() error

type Config struct {
	Level      string `json:"level"`      // 日志级别，有效值包括 "debug"、"info"、"warn"、"error" 和 "fatal"
	Filename   string `json:"filename"`   // 日志文件的位置
	MaxSize    int    `json:"maxSize"`    // 单个日志文件的最大大小，单位为兆字节
	MaxBackups int    `json:"maxBackups"` // 保留的旧日志文件的最大数量
	MaxAge     int    `json:"maxAge"`     // 保留旧日志文件的最大天数
	Compress   bool   `json:"compress"`   // 是否压缩旧的日志文件
	Stdout     bool   `json:"stdout"`     // 是否将日志输出到标准输出（控制台）
}

var log *zap.Logger

func Init(config Config) (LogSync, error) {
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

	if config.Stdout {
		consoleWriters = append(consoleWriters, zapcore.AddSync(os.Stdout))
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderCfg.EncodeDuration = zapcore.StringDurationEncoder
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	fileEncoder := zapcore.NewJSONEncoder(encoderCfg)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, err
	}

	fileCore := zapcore.NewCore(fileEncoder, zapcore.NewMultiWriteSyncer(fileWriters...), level)

	consoleLevel := zapcore.InfoLevel
	if level >= consoleLevel {
		consoleLevel = level
	}
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.NewMultiWriteSyncer(consoleWriters...), consoleLevel)

	core := zapcore.NewTee(fileCore, consoleCore)
	log = zap.New(core, zap.AddCaller(), zap.Development(), zap.AddCallerSkip(1))

	return sync, nil
}

func Info(msg string, fields ...zapcore.Field) {
	log.Info(msg, fields...)
}

func Warn(msg string, fields ...zapcore.Field) {
	log.Warn(msg, fields...)
}

func Error(msg string, fields ...zapcore.Field) {
	log.Error(msg, fields...)
}

func DPanic(msg string, fields ...zapcore.Field) {
	log.DPanic(msg, fields...)
}

func Panic(msg string, fields ...zapcore.Field) {
	log.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zapcore.Field) {
	log.Fatal(msg, fields...)
}

func Debug(msg string, fields ...zapcore.Field) {
	log.Debug(msg, fields...)
}

func sync() error {
	return log.Sync()
}
