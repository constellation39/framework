package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrWorkerStopped = errors.New("worker pool has been stopped")
	ErrTaskTimeout   = errors.New("task execution timeout")
	ErrPoolFull      = errors.New("worker pool queue is full")
	ErrInvalidConfig = errors.New("invalid worker pool configuration")
)

// Options 定义了工作池的配置选项。
type Options struct {
	// MinWorkers: 最小工作线程数量。
	MinWorkers int
	// MaxWorkers: 最大工作线程数量。
	MaxWorkers int
	// QueueSize: 任务队列大小。
	QueueSize int
	// MetricsEnabled: 是否启用指标统计。
	MetricsEnabled bool
	// ShutdownTimeout: 工作池关闭超时时间。
	ShutdownTimeout time.Duration
}

// DefaultOptions 返回默认配置
func DefaultOptions() Options {
	return Options{
		MinWorkers:      1,
		MaxWorkers:      10,
		QueueSize:       100,
		MetricsEnabled:  true,
		ShutdownTimeout: 30 * time.Second,
	}
}

type Task[T any] struct {
	// Fn: 泛型任务的执行函数。
	Fn func() T
	// resultCh: 用于发送任务结果的通道。
	resultCh chan<- Result[T]
}

// Result 封装任务执行结果（泛型方案）
type Result[T any] struct {
	Value T
	Err   error
}

// Worker 定义了工作池的接口
type Worker[T any] interface {
	// Submit 提交任务到工作池
	Submit(ctx context.Context, task func() T, ch chan<- Result[T]) error
	// Start 启动工作池
	Start() error
	// Stop 停止工作池
	Stop() error
	// Metrics 返回工作池的指标
	Metrics() Metrics
	// Scale 动态调整工作池大小
	Scale(delta int) error
}

// Metrics 工作池指标
type Metrics struct {
	TaskCount      *atomic.Int64
	ActiveTasks    *atomic.Int32
	QueueLength    *atomic.Int64
	RejectedTasks  *atomic.Int64
	TimeoutTasks   *atomic.Int64
	CompletedTasks *atomic.Int64
	ErrorCount     *atomic.Int64
	ActiveWorkers  *atomic.Int32
	AverageTime    atomic.Value // stores time.Duration
}

// NewMetrics 创建新的指标实例
func NewMetrics() Metrics {
	m := Metrics{
		TaskCount:      &atomic.Int64{},
		ActiveTasks:    &atomic.Int32{},
		QueueLength:    &atomic.Int64{},
		RejectedTasks:  &atomic.Int64{},
		TimeoutTasks:   &atomic.Int64{},
		CompletedTasks: &atomic.Int64{},
		ErrorCount:     &atomic.Int64{},
		ActiveWorkers:  &atomic.Int32{},
	}
	m.AverageTime.Store(time.Duration(0))
	return m
}
