package statemachine

import "fmt"

type Task func()

// State 接口定义了状态的行为
type State interface {
	Enter(context *StateContext)
	Exit(context *StateContext)
	AddTasks(tasks []Task)

	name() string
}

type BaseState struct {
	Name  string
	tasks []Task
}

func (s *BaseState) Enter(context *StateContext) {
	fmt.Printf("Entering %s\n", s.Name)
	for _, task := range s.tasks {
		context.AddTask(task)
	}
}

func (s *BaseState) Exit(context *StateContext) {
	fmt.Printf("Exiting %s\n", s.Name)
}

func (s *BaseState) AddTasks(tasks []Task) {
	s.tasks = append(s.tasks, tasks...)
}

func (s *BaseState) name() string {
	return s.Name
}
