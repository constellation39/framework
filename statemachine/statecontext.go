package statemachine

import "sync"

// StateContext 维护当前状态和状态机的上下文
type StateContext struct {
	currentState State
	tasks        chan Task
	wg           sync.WaitGroup
	stateChan    chan State
	quit         chan struct{}
}

// NewStateContext 创建一个新的状态上下文
func NewStateContext(initialState State) *StateContext {
	ctx := &StateContext{
		currentState: initialState,
		tasks:        make(chan Task),
		stateChan:    make(chan State),
		quit:         make(chan struct{}),
	}
	go ctx.worker()
	return ctx
}

func (sc *StateContext) worker() {
	for {
		select {
		case task := <-sc.tasks:
			task()
			sc.wg.Done()
		case newState := <-sc.stateChan:
			sc.currentState.Exit(sc)
			sc.currentState = newState
			sc.currentState.Enter(sc)
		case <-sc.quit:
			return
		}
	}
}

func (sc *StateContext) AddTask(task Task) {
	sc.wg.Add(1)
	go func() {
		sc.tasks <- task
	}()
}

func (sc *StateContext) WaitAndTransition(newState State) {
	sc.wg.Wait()
	sc.stateChan <- newState
}

func (sc *StateContext) Stop() {
	close(sc.quit)
}
