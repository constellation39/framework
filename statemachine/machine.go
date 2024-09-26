package statemachine

import "fmt"

type StateMachine struct {
	context *StateContext
}

func NewStateMachine(initialState State) *StateMachine {
	return &StateMachine{
		context: NewStateContext(initialState),
	}
}

func (sm *StateMachine) TransitionTo(newState State) {
	fmt.Printf("Transitioning from %s to %s\n", sm.context.currentState.name(), newState.name())
	sm.context.WaitAndTransition(newState)
}

func (sm *StateMachine) CurrentState() State {
	return sm.context.currentState
}

func (sm *StateMachine) Run() {
	sm.context.currentState.Enter(sm.context)
}

func (sm *StateMachine) Stop() {
	sm.context.Stop()
}
