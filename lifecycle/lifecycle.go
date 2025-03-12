// Package lifecycle 提供应用程序生命周期管理，支持优雅启动和关闭。
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

// DefaultOptions 返回默认配置
func DefaultOptions() Options {
	return Options{
		ShutdownTimeout: 30 * time.Second, // 默认30秒超时
	}
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
		done:         make(chan struct{}),
		opts:         opts,
		cleanupHooks: []func(){},
	}

	// 监听系统信号
	go lc.listenForSignals()

	return lc
}

// Context 返回生命周期的上下文
func (l *Lifecycle) Context() context.Context {
	return l.ctx
}

// AddCleanupHook 添加清理钩子
func (l *Lifecycle) AddCleanupHook(hook func()) {
	l.cleanupHooks = append(l.cleanupHooks, hook)
}

// listenForSignals 监听系统信号
func (l *Lifecycle) listenForSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		fmt.Printf("Received signal: %v\n", sig)
		l.Shutdown()
	case <-l.ctx.Done():
		// 上下文已取消，无需额外操作
	}
}

// executeCleanupHooks 执行所有清理钩子
func (l *Lifecycle) executeCleanupHooks() {
	for _, hook := range l.cleanupHooks {
		hook()
	}
}

// Go 启动一个受管理的goroutine
func (l *Lifecycle) Go(f func(ctx context.Context) error) <-chan error {
	errChan := make(chan error, 1)

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer close(errChan)
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic occurred: %v", r)
			}
		}()

		if err := f(l.ctx); err != nil {
			errChan <- err
		}
	}()

	return errChan
}

// Wait 等待生命周期结束
func (l *Lifecycle) Wait() {
	<-l.done
}

// Shutdown 触发优雅关闭流程
func (l *Lifecycle) Shutdown() {
	l.shutdownOnce.Do(func() {
		// 取消上下文
		l.cancel()

		// 等待所有goroutine完成或超时
		waitCh := make(chan struct{})
		go func() {
			l.wg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			// 所有goroutine已完成
		case <-time.After(l.opts.ShutdownTimeout):
			// 超时
		}

		// 执行清理钩子
		l.executeCleanupHooks()

		// 标记完成
		close(l.done)
	})
}
