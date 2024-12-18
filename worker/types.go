package worker

import (
	"context"
	"errors"
	"runtime"
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
	// WorkerSize: 工作线程数量。
	WorkerSize int
	// QueueSize: 任务队列大小。
	QueueSize int
	// ShutdownTimeout: 工作池关闭超时时间。
	ShutdownTimeout time.Duration
}

// DefaultOptions 返回默认配置
func DefaultOptions() Options {
	return Options{
		WorkerSize:      runtime.NumCPU(),
		QueueSize:       runtime.NumCPU() * 10,
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
	Metrics() *Metrics
	// Scale 动态调整工作池大小
	Scale(delta int) error
}
