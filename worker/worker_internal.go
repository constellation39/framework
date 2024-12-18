package worker

import (
	"errors"
	"fmt"
	"time"
)

// startWorker 启动一个新的工作线程。
func (w *worker[T]) startWorker() {
	w.wg.Add(1)
	w.metrics.ActiveWorkers.Add(1)
	go func() {
		defer w.wg.Done()
		defer w.metrics.ActiveWorkers.Add(-1)
		w.runWorker()
	}()
}

// runWorker 运行工作线程。
func (w *worker[T]) runWorker() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.quit:
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}

			if err := w.executeTask(task); err != nil {
				w.metrics.ErrorCount.Add(1)
			}
		}
	} // 添加缺失的右大括号
}

func (w *worker[T]) executeTask(task Task[T]) error {
	start := time.Now()

	resultCh := make(chan T, 1)
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("task panicked: %v", r)
			}
		}()
		resultCh <- task.Fn()
	}()

	select {
	case result := <-resultCh:
		w.handleTaskCompletion(task, result, nil, start)
		return nil
	case err := <-errCh:
		w.handleTaskCompletion(task, *new(T), err, start)
		return err
	case <-w.ctx.Done():
		err := fmt.Errorf("task execution canceled: %v", w.ctx.Err())
		w.handleTaskCompletion(task, *new(T), err, start)
		return err
	}
}

func (w *worker[T]) handleTaskCompletion(task Task[T], result T, err error, start time.Time) {
	w.metrics.ActiveTasks.Add(-1)
	w.metrics.QueueLength.Add(-1)

	if err != nil {
		w.metrics.ErrorCount.Add(1)
		if task.resultCh != nil {
			task.resultCh <- Result[T]{Err: err}
		}
	} else {
		w.metrics.CompletedTasks.Add(1)
		if task.resultCh != nil {
			task.resultCh <- Result[T]{Value: result}
		}
	}

	w.updateMetrics(time.Since(start))
}

// updateMetrics 更新工作池的平均执行时间指标。
func (w *worker[T]) updateMetrics(duration time.Duration) {
	const alpha = 0.1
	oldAvg := w.metrics.AverageTime.Load().(time.Duration)
	newAvg := time.Duration(float64(oldAvg)*(1-alpha) + float64(duration)*alpha)
	w.metrics.AverageTime.Store(newAvg)
}

// autoScale 自动扩缩容线程数。
func (w *worker[T]) autoScale() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.adjustWorkerCount()
		}
	}
}

func (w *worker[T]) adjustWorkerCount() {
	queueLen := w.metrics.QueueLength.Load()
	activeWorkers := w.metrics.ActiveWorkers.Load()
	completedTasks := w.metrics.CompletedTasks.Load()

	// 计算任务处理率
	processingRate := float64(completedTasks) / float64(activeWorkers)

	switch {
	// 队列积压且处理率高，增加工作者
	case queueLen > int64(activeWorkers*2) &&
		processingRate >= 0.8 &&
		int(activeWorkers) < w.opts.MaxWorkers:
		w.Scale(min(w.opts.MaxWorkers-int(activeWorkers), 2))

	// 队列为空且工作者空闲，减少工作者
	case queueLen == 0 &&
		processingRate < 0.2 &&
		activeWorkers > int32(w.opts.MinWorkers):
		w.Scale(-1)
	}
}

// validateOptions 校验工作池配置选项。
func validateOptions(opts Options) error {
	if opts.MaxWorkers < opts.MinWorkers {
		return errors.New("MaxWorkers must be greater than or equal to MinWorkers")
	}
	if opts.QueueSize <= 0 {
		return errors.New("QueueSize must be greater than 0")
	}
	return nil
}
