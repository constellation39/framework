package lifecycle

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

/**********************************************************************
* 公共 API
**********************************************************************/

// AppContext 返回“运行期” ctx；第一次收到信号或手动 BeginShutdown 时会被取消。
func AppContext() context.Context { return def.appCtx }

// ShutdownContext 返回“停机期” ctx；
// 未进入停机流程前等同于 context.Background()。
func ShutdownContext() context.Context { return def.shutdownCtx() }

// Register 在优雅停机阶段要执行的清理回调。
// 回调会在单独 goroutine 中执行；ctx == ShutdownContext()。
func Register(fn func(context.Context) error) { def.register(fn) }

// SetTimeout 修改优雅停机超时时间（默认 15s）；只能在 Running 状态下调用。
func SetTimeout(d time.Duration) { def.setTimeout(d) }

// BeginShutdown 手动触发优雅停机（通常不需要调用，信号监听会自动触发）。
func BeginShutdown() { def.beginShutdown() }

// Wait 阻塞直到停机流程“完成”(所有 Hook 跑完或超时)。
func Wait() { <-def.done }

// IsRunning / IsShuttingDown / IsStopped = 当前状态观察
func IsRunning() bool      { return atomic.LoadInt32(&def.state) == stateRunning }
func IsShuttingDown() bool { return atomic.LoadInt32(&def.state) == stateShutting }
func IsStopped() bool      { return atomic.LoadInt32(&def.state) == stateStopped }

/**********************************************************************
* 默认单例 manager
**********************************************************************/

var def = newManager(defaultTimeout)

const (
	defaultTimeout = 15 * time.Second

	stateRunning int32 = iota
	stateShutting
	stateStopped
)

type manager struct {
	// ---- 生命周期 ctx ----
	appCtx    context.Context // 运行期 ctx（被信号/手动关闭时 cancel）
	appCancel context.CancelFunc
	sdCtx     context.Context
	sdCancel  context.CancelFunc

	// ---- 钩子 ----
	mu    sync.Mutex
	hooks []func(context.Context) error
	wg    sync.WaitGroup

	// ---- 其它 ----
	state   int32         // atomic
	timeout time.Duration // 优雅停机超时
	done    chan struct{} // 彻底完成时关闭
	once    sync.Once     // 保证 BeginShutdown 只执行一次
}

func newManager(timeout time.Duration) *manager {
	m := &manager{
		timeout: timeout,
		state:   stateRunning,
		done:    make(chan struct{}),
	}

	m.appCtx, m.appCancel = context.WithCancel(context.Background())

	// 信号监听
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			sig := <-sigCh
			if atomic.LoadInt32(&m.state) == stateRunning {
				log.Printf("[lifecycle] receive %s, begin graceful shutdown", sig)
				m.beginShutdown()
			} else {
				log.Printf("[lifecycle] receive %s again, force exit", sig)
				os.Exit(1)
			}
		}
	}()
	return m
}

/**********************************************************************
* manager 逻辑
**********************************************************************/

func (m *manager) shutdownCtx() context.Context {
	// 尚未进入停机态时返回背景 ctx，以保证永不为 nil
	if atomic.LoadInt32(&m.state) == stateRunning {
		return context.Background()
	}
	return m.sdCtx
}

func (m *manager) setTimeout(d time.Duration) {
	if atomic.LoadInt32(&m.state) != stateRunning {
		return // 只允许在正常运行时改
	}
	m.timeout = d
}

func (m *manager) register(fn func(context.Context) error) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// 若已进入停机流程，直接异步执行到当前 ctx 中
	if atomic.LoadInt32(&m.state) != stateRunning {
		go fn(m.shutdownCtx())
		return
	}
	m.hooks = append(m.hooks, fn)
}

func (m *manager) beginShutdown() {
	// 只执行一次
	m.once.Do(func() {
		if !atomic.CompareAndSwapInt32(&m.state, stateRunning, stateShutting) {
			return
		}

		m.appCancel()

		m.sdCtx, m.sdCancel = context.WithTimeout(context.Background(), m.timeout)

		go m.runHooks()

		go func() {
			<-m.sdCtx.Done()
			m.complete()
		}()
	})
}

func (m *manager) runHooks() {
	m.mu.Lock()
	hooks := make([]func(context.Context) error, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.Unlock()

	for _, h := range hooks {
		m.wg.Add(1)
		go func(fn func(context.Context) error) {
			defer m.wg.Done()
			_ = fn(m.sdCtx)
		}(h)
	}

	// 等待全部 hook 结束
	m.wg.Wait()
	m.complete()
}

func (m *manager) complete() {
	if !atomic.CompareAndSwapInt32(&m.state, stateShutting, stateStopped) {
		return
	}
	if m.sdCancel != nil {
		m.sdCancel()
	}
	close(m.done)
}
