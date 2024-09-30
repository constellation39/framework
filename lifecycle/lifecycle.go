package lifecycle

import (
	"context"
	"framework/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var life *Lifecycle

func init() {
	life = New()
}

type Lifecycle struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	done   chan struct{}
}

func New() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	lc := &Lifecycle{
		ctx:    ctx,
		cancel: cancel,
		wg:     sync.WaitGroup{},
		done:   make(chan struct{}),
	}

	go lc.listenForShutdown()

	return lc
}

func (l *Lifecycle) Context() context.Context {
	return l.ctx
}

func (l *Lifecycle) listenForShutdown() {
	sigChan := make(chan os.Signal, 1)
	defer signal.Stop(sigChan) // 确保在退出时停止信号通知
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case _, ok := <-sigChan:
		if !ok {
			return
		}
		l.cancel()
	}

	timeout := time.After(2 * time.Second)
	doneChan := make(chan struct{})

	go func() {
		l.wg.Wait()
		close(doneChan)
	}()

	logger.Info("waiting for goroutines to finish")
	select {
	case <-doneChan:
	case <-timeout:
		logger.Error("timeout waiting for goroutines to finish")
	}

	close(l.done)
}

func (l *Lifecycle) Go(f func(ctx context.Context)) {
	l.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic in lifecycle.Go", zap.Reflect("r", r))
			}
		}()
		defer l.wg.Done()
		f(l.ctx)
	}()
}

func (l *Lifecycle) Wait() {
	<-l.done
}

func Context() context.Context {
	return life.Context()
}

func Go(f func(ctx context.Context)) {
	life.Go(f)
}

func Wait() {
	life.Wait()
}
