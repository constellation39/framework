package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// worker 表示工作池的实现。

type worker[T any] struct {
	// opts 工作池的配置选项。
	opts Options
	// tasks 任务通道，用于分发任务。
	tasks chan Task[T]
	// wg 等待组，用于等待所有工作线程完成。
	wg sync.WaitGroup
	// quit 关闭信号通道。
	quit chan struct{}
	// ctx, cancel 用于控制工作池生命周期的上下文。
	ctx    context.Context
	cancel context.CancelFunc
	// running 表示工作池是否正在运行的状态。
	running atomic.Bool
	// metrics 存储工作池的性能指标。
	metrics *Metrics
}

// NewWorker 创建一个新的工作池。
func NewWorker[T any](ctx context.Context, opts Options) (Worker[T], error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}
	w := &worker[T]{
		opts:    opts,
		quit:    make(chan struct{}),
		metrics: NewMetrics(),
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	return w, nil
}

func (w *worker[T]) Submit(ctx context.Context, task func() T, ch chan<- Result[T]) error {
	if !w.running.Load() {
		ch <- Result[T]{Err: ErrWorkerStopped}
		return ErrWorkerStopped
	}

	wrappedTask := Task[T]{
		Fn:       task,
		resultCh: ch,
	}

	if w.tasks == nil {
		return errors.New("worker tasks channel not initialized, call Start() first")
	}
	select {
	case <-ctx.Done():
		ch <- Result[T]{Err: ctx.Err()}
		return ctx.Err()
	case <-w.ctx.Done():
		ch <- Result[T]{Err: ErrWorkerStopped}
		return ErrWorkerStopped
	case w.tasks <- wrappedTask:
		w.metrics.IncrementQueueLength(1)
		w.metrics.IncrementActiveTasks(1)
		return nil
	}
}

// Start 启动工作池，初始化任务通道并创建最小数量的工作线程。
func (w *worker[T]) Start() error {
	if !w.running.CompareAndSwap(false, true) {
		return errors.New("worker pool is already running")
	}

	w.tasks = make(chan Task[T], w.opts.QueueSize)

	for i := 0; i < w.opts.WorkerSize; i++ {
		w.startWorker()
	}
	return nil
}

// Stop 优雅地停止工作池，等待所有任务完成。
func (w *worker[T]) Stop() error {
	if !w.running.CompareAndSwap(true, false) {
		return nil
	}

	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	timeout := time.After(w.opts.ShutdownTimeout)

	select {
	case <-done:
	case <-timeout:
		w.metrics.IncrementErrorCount(int64(w.metrics.GetActiveTasks()))
	}

	close(w.quit)
	close(w.tasks)

	return nil
}

func (w *worker[T]) Metrics() *Metrics {
	return w.metrics
}

func (w *worker[T]) Scale(delta int) error {
	if !w.running.Load() {
		return ErrWorkerStopped
	}

	currentWorkers := w.metrics.GetActiveWorkers()
	newCount := int(currentWorkers) + delta

	if newCount < 0 {
		return errors.New("cannot scale down below 0 workers")
	}

	if delta > 0 {
		for i := 0; i < delta; i++ {
			w.startWorker()
		}
	} else {
		for i := 0; i < -delta; i++ {
			select {
			case w.quit <- struct{}{}:
			case <-time.After(10 * time.Millisecond):
				return errors.New("failed to scale down workers")
			}
		}
	}

	return nil
}
