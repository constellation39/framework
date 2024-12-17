package lifecycle

import (
	"context"
	"fmt"
	"framework/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Options 配置选项
type Options struct {
	// 关闭超时时间
	ShutdownTimeout time.Duration
	// 错误处理函数
	ErrorHandler func(error)
	// 信号处理函数
	SignalHandler func(os.Signal)
	// 日志接口
	Logger logger.Logger
}

// Lifecycle 生命周期管理器
type Lifecycle struct {
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	done         chan struct{}
	opts         Options
	cleanupHooks []func() error
	errors       chan error
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
		cleanupHooks: make([]func() error, 0),
		errors:       make(chan error, 100),
	}

	go lc.listenForShutdown()
	go lc.handleErrors()

	return lc
}

// Context 返回context
func (l *Lifecycle) Context() context.Context {
	return l.ctx
}

// AddCleanupHook 添加清理钩子
func (l *Lifecycle) AddCleanupHook(hook func() error) {
	l.cleanupHooks = append(l.cleanupHooks, hook)
}

// handleErrors 处理错误channel中的错误
func (l *Lifecycle) handleErrors() {
	for err := range l.errors {
		l.logError(err)
	}
}

// logError 记录错误
func (l *Lifecycle) logError(err error) {
	if l.opts.ErrorHandler != nil {
		l.opts.ErrorHandler(err)
	}
	l.opts.Logger.Error("goroutine error", zap.Error(err))
}

// listenForShutdown 监听关闭信号
func (l *Lifecycle) listenForShutdown() {
	sig := l.waitForSignal()
	if sig != nil {
		l.opts.Logger.Debug("received signal", zap.String("signal", sig.String()))
		if l.opts.SignalHandler != nil {
			l.opts.SignalHandler(sig)
		}
	} else {
		l.opts.Logger.Debug("context cancelled")
	}
	l.shutdown()
}

// waitForSignal 等待关闭信号
func (l *Lifecycle) waitForSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case s := <-sigChan:
		return s
	case <-l.ctx.Done():
		return nil
	}
}

// shutdown 执行关闭流程
func (l *Lifecycle) shutdown() {
	l.shutdownOnce.Do(func() {
		l.cancel()

		// 执行清理钩子
		l.executeCleanupHooks()

		// 等待所有goroutine完成或超时
		l.waitForGoroutines()

		close(l.errors)
		close(l.done)
	})
}

// executeCleanupHooks 执行清理钩子
func (l *Lifecycle) executeCleanupHooks() {
	for _, hook := range l.cleanupHooks {
		if err := hook(); err != nil {
			l.logError(err)
		}
	}
}

// waitForGoroutines 等待所有goroutine完成或超时
func (l *Lifecycle) waitForGoroutines() {
	doneChan := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(doneChan)
	}()

	l.opts.Logger.Debug("waiting for goroutines to finish")
	select {
	case <-doneChan:
		l.opts.Logger.Debug("all goroutines finished")
	case <-time.After(l.opts.ShutdownTimeout):
		l.opts.Logger.Error("timeout waiting for goroutines to finish")
	}
}

// Go 启动一个受管理的goroutine
func (l *Lifecycle) Go(f func(ctx context.Context) error) {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic recovered: %v", r)
				l.errors <- err
			}
		}()

		if err := f(l.ctx); err != nil {
			l.errors <- err
		}
	}()
}

// Wait 等待生命周期结束
func (l *Lifecycle) Wait() {
	<-l.done
}

// Shutdown 主动触发关闭
func (l *Lifecycle) Shutdown() {
	l.cancel()
	l.Wait()
}
