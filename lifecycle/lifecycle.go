package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Options 配置选项
type Options struct {
	ShutdownTimeout time.Duration
}

// Lifecycle 生命周期管理器
type Lifecycle struct {
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	done         chan struct{}
	opts         Options
	cleanupHooks []func()
	shutdownOnce sync.Once
}

// New 创建新的生命周期管理器
func New(opts Options) *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())

	lc := &Lifecycle{
		ctx:          ctx,
		cancel:       cancel,
		wg:           sync.WaitGroup{},
		done:         make(chan struct{}),
		opts:         opts,
		cleanupHooks: make([]func(), 0),
	}

	go lc.listenForSignalsAndShutdown()

	return lc
}

// Context 返回context
func (l *Lifecycle) Context() context.Context {
	return l.ctx
}

// AddCleanupHook 添加清理钩子
func (l *Lifecycle) AddCleanupHook(hook func()) {
	l.cleanupHooks = append(l.cleanupHooks, hook)
}

// listenForSignalsAndShutdown 监听来自系统的信号和上下文的取消
func (l *Lifecycle) listenForSignalsAndShutdown() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		fmt.Printf("Received signal: %v\n", sig)
		l.shutdown()
	case <-l.ctx.Done():
		l.shutdown()
	}
}

// shutdown 执行关闭流程
func (l *Lifecycle) shutdown() {
	l.shutdownOnce.Do(func() {
		l.cancel()
		l.waitForGoroutines()
		l.executeCleanupHooks()
		close(l.done)
	})
}

// executeCleanupHooks 执行清理钩子并返回错误
func (l *Lifecycle) executeCleanupHooks() {
	for _, hook := range l.cleanupHooks {
		hook()
	}
}

// waitForGoroutines 等待所有goroutine完成或超时
func (l *Lifecycle) waitForGoroutines() {
	doneChan := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		// 所有 goroutine 已完成
	case <-time.After(l.opts.ShutdownTimeout):
		// 超时等待 goroutine 完成
	}
}

// Go 启动一个受管理的goroutine并返回可能的错误
func (l *Lifecycle) Go(f func(ctx context.Context) error) <-chan error {
	errChan := make(chan error, 1)
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				// 在这里记录 panic 错误
				errChan <- fmt.Errorf("panic occurred: %v", r)
			}
		}()

		if err := f(l.ctx); err != nil {
			errChan <- err // 将错误传递给外部
		}
		close(errChan)
	}()
	return errChan
}

// Wait 等待生命周期结束
func (l *Lifecycle) Wait() {
	<-l.done
}

// Shutdown 主动触发关闭
func (l *Lifecycle) Shutdown() {
	l.shutdown()
}
