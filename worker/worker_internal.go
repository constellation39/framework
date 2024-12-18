package worker

import (
	"errors"
	"fmt"
	"time"
)

// startWorker 启动一个新的工作线程。
func (w *worker[T]) startWorker() {
	w.wg.Add(1)
	w.metrics.IncrementActiveWorkers(1)
	go func() {
		defer w.wg.Done()
		defer w.metrics.IncrementActiveWorkers(-1)
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
				w.metrics.IncrementErrorCount(1)
			}
		}
	}
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
		err := fmt.Errorf("%w: %v", ErrTaskCancelled, w.ctx.Err())
		w.handleTaskCompletion(task, *new(T), err, start)
		return err
	}
}

func (w *worker[T]) handleTaskCompletion(task Task[T], result T, err error, start time.Time) {
	w.metrics.IncrementActiveTasks(-1)
	w.metrics.IncrementQueueLength(-1)

	if err != nil {
		w.metrics.IncrementErrorCount(1)
		if task.resultCh != nil {
			task.resultCh <- Result[T]{Err: err}
		}
	} else {
		w.metrics.IncrementCompletedTasks(1)
		if task.resultCh != nil {
			task.resultCh <- Result[T]{Value: result}
		}
	}

	w.metrics.UpdateAverageTime(time.Since(start))
}

// validateOptions 校验工作池配置选项。
func validateOptions(opts Options) error {
	if opts.WorkerSize <= 0 {
		return errors.New("WorkerSize must be greater than 0")
	}
	if opts.QueueSize <= 0 {
		return errors.New("QueueSize must be greater than 0")
	}
	return nil
}
