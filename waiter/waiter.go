package waitercontext

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type WaiterContext struct {
	parent     context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	err        atomic.Value
	done       chan struct{}
	closeOnce  sync.Once
}

func WitchWaiter(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	wc := &WaiterContext{
		parent:     ctx,
		cancelFunc: cancel,
		done:       make(chan struct{}),
	}
	go wc.watchParent()
	return wc, wc.CancelFunc
}

func (wc *WaiterContext) watchParent() {
	<-wc.parent.Done()
	wc.closeOnce.Do(func() {
		close(wc.done)
	})
	wc.err.Store(wc.parent.Err())
}

func (wc *WaiterContext) Deadline() (deadline time.Time, ok bool) {
	return wc.parent.Deadline()
}

func (wc *WaiterContext) Done() <-chan struct{} {
	return wc.done
}

func (wc *WaiterContext) Err() error {
	if err := wc.err.Load(); err != nil {
		return err.(error)
	}
	return nil
}

func (wc *WaiterContext) Value(key interface{}) interface{} {
	return wc.parent.Value(key)
}

func (wc *WaiterContext) CancelFunc() {
	wc.cancelFunc()
	wc.wg.Wait()
}
