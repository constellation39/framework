package logger

import (
	"sync/atomic"

	"go.uber.org/zap"
)

var (
	globalLogger atomic.Value
	nopLogger    = zap.NewNop()
)

// Init 初始化全局日志
func Init(opts ...Option) error {
	logger, err := New(opts...)
	if err != nil {
		return err
	}

	SetGlobal(logger)
	return nil
}

// MustInit 初始化全局日志，失败则 panic
func MustInit(opts ...Option) {
	if err := Init(opts...); err != nil {
		panic(err)
	}
}

// SetGlobal 设置全局日志实例
func SetGlobal(logger *Logger) {
	globalLogger.Store(logger)
	// 同时设置 zap 的全局 logger
	zap.ReplaceGlobals(logger.Logger)
}

// GetGlobal 获取全局日志实例
func GetGlobal() *Logger {
	if v := globalLogger.Load(); v != nil {
		return v.(*Logger)
	}
	return nil
}

// L 返回全局 Logger（如果未初始化则返回 nop logger）
func L() *zap.Logger {
	if v := globalLogger.Load(); v != nil {
		return v.(*Logger).Logger
	}
	return nopLogger
}

// S 返回全局 SugaredLogger（如果未初始化则返回 nop logger）
func S() *zap.SugaredLogger {
	if v := globalLogger.Load(); v != nil {
		return v.(*Logger).Sugar()
	}
	return nopLogger.Sugar()
}

// Sync 同步全局日志
func Sync() error {
	if logger := GetGlobal(); logger != nil {
		return logger.Sync()
	}
	return nil
}

// Close 关闭全局日志
func Close() error {
	if logger := GetGlobal(); logger != nil {
		return logger.Close()
	}
	return nil
}

// 便捷方法 - 直接使用全局 logger

// Debug 输出 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

// Info 输出 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

// Warn 输出 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

// Error 输出 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

// DPanic 输出 DPanic 级别日志
func DPanic(msg string, fields ...zap.Field) {
	L().DPanic(msg, fields...)
}

// Panic 输出 Panic 级别日志
func Panic(msg string, fields ...zap.Field) {
	L().Panic(msg, fields...)
}

// Fatal 输出 Fatal 级别日志
func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

// Debugf 格式化输出 Debug 级别日志
func Debugf(template string, args ...interface{}) {
	S().Debugf(template, args...)
}

// Infof 格式化输出 Info 级别日志
func Infof(template string, args ...interface{}) {
	S().Infof(template, args...)
}

// Warnf 格式化输出 Warn 级别日志
func Warnf(template string, args ...interface{}) {
	S().Warnf(template, args...)
}

// Errorf 格式化输出 Error 级别日志
func Errorf(template string, args ...interface{}) {
	S().Errorf(template, args...)
}

// DPanicf 格式化输出 DPanic 级别日志
func DPanicf(template string, args ...interface{}) {
	S().DPanicf(template, args...)
}

// Panicf 格式化输出 Panic 级别日志
func Panicf(template string, args ...interface{}) {
	S().Panicf(template, args...)
}

// Fatalf 格式化输出 Fatal 级别日志
func Fatalf(template string, args ...interface{}) {
	S().Fatalf(template, args...)
}

// With 创建带有字段的 logger
func With(fields ...zap.Field) *zap.Logger {
	return L().With(fields...)
}

// WithOptions 使用选项创建 logger
func WithOptions(opts ...zap.Option) *zap.Logger {
	return L().WithOptions(opts...)
}
