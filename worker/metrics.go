package worker

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Metrics struct {
	// taskCount 记录提交的任务总数。
	taskCount *atomic.Int64
	// activeTasks 当前正在执行的任务数。
	activeTasks *atomic.Int32
	// queueLength 当前任务队列的长度。
	queueLength *atomic.Int64
	// rejectedTasks 被拒绝执行的任务数（例如因队列已满）。
	rejectedTasks *atomic.Int64
	// timeoutTasks 执行超时的任务数。
	timeoutTasks *atomic.Int64
	// completedTasks 已完成执行的任务数。
	completedTasks *atomic.Int64
	// errorCount 记录执行过程中发生错误的任务数。
	errorCount *atomic.Int64
	// activeWorkers 当前正在运行的工作线程数。
	activeWorkers *atomic.Int32
	// averageTime 平均任务执行时间（单位：时间段）。
	averageTime atomic.Value // stores time.Duration
}

func NewMetrics() *Metrics {
	m := &Metrics{
		taskCount:      &atomic.Int64{},
		activeTasks:    &atomic.Int32{},
		queueLength:    &atomic.Int64{},
		rejectedTasks:  &atomic.Int64{},
		timeoutTasks:   &atomic.Int64{},
		completedTasks: &atomic.Int64{},
		errorCount:     &atomic.Int64{},
		activeWorkers:  &atomic.Int32{},
	}
	m.averageTime.Store(time.Duration(0))

	return m
}

// IncrementCompletedTasks 增加完成任务计数
func (m *Metrics) IncrementCompletedTasks(delta int64) {
	m.completedTasks.Add(delta)
}

func (m *Metrics) IncrementTaskCount(delta int64) {
	m.taskCount.Add(delta)
}

func (m *Metrics) IncrementActiveTasks(delta int32) {
	m.activeTasks.Add(delta)
}

func (m *Metrics) IncrementQueueLength(delta int64) {
	m.queueLength.Add(delta)
}

func (m *Metrics) IncrementRejectedTasks(delta int64) {
	m.rejectedTasks.Add(delta)
}

func (m *Metrics) IncrementTimeoutTasks(delta int64) {
	m.timeoutTasks.Add(delta)
}

func (m *Metrics) IncrementErrorCount(delta int64) {
	m.errorCount.Add(delta)
}

func (m *Metrics) IncrementActiveWorkers(delta int32) {
	m.activeWorkers.Add(delta)
}

func (m *Metrics) GetTaskCount() int64 {
	return m.taskCount.Load()
}

func (m *Metrics) GetActiveTasks() int32 {
	return m.activeTasks.Load()
}

func (m *Metrics) GetQueueLength() int64 {
	return m.queueLength.Load()
}

func (m *Metrics) GetRejectedTasks() int64 {
	return m.rejectedTasks.Load()
}

func (m *Metrics) GetTimeoutTasks() int64 {
	return m.timeoutTasks.Load()
}

func (m *Metrics) GetCompletedTasks() int64 {
	return m.completedTasks.Load()
}

func (m *Metrics) GetErrorCount() int64 {
	return m.errorCount.Load()
}

func (m *Metrics) GetActiveWorkers() int32 {
	return m.activeWorkers.Load()
}

func (m *Metrics) GetAverageTime() time.Duration {
	return m.averageTime.Load().(time.Duration)
}

func (m *Metrics) UpdateAverageTime(duration time.Duration) {
	const alpha = 0.1
	oldAvg := m.GetAverageTime()
	newAvg := time.Duration(float64(oldAvg)*(1-alpha) + float64(duration)*alpha)
	m.averageTime.Store(newAvg)
}

// Reset 重置所有计数
func (m *Metrics) Reset() {
	// 重置基础计数器
	m.taskCount.Store(0)
	m.activeTasks.Store(0)
	m.queueLength.Store(0)
	m.rejectedTasks.Store(0)
	m.timeoutTasks.Store(0)
	m.completedTasks.Store(0)
	m.errorCount.Store(0)
	m.activeWorkers.Store(0)
	m.averageTime.Store(time.Duration(0))
}

// String 输出更详细的指标信息，包括窗口统计
func (m *Metrics) String() string {
	base := fmt.Sprintf(
		"Metrics:\n"+
			"- Task Count: %d\n"+
			"- Active Tasks: %d\n"+
			"- Queue Length: %d\n"+
			"- Rejected Tasks: %d\n"+
			"- Timeout Tasks: %d\n"+
			"- Completed Tasks: %d\n"+
			"- Error Count: %d\n"+
			"- Active Workers: %d\n"+
			"- Average Task Time: %v\n",
		m.GetTaskCount(),
		m.GetActiveTasks(),
		m.GetQueueLength(),
		m.GetRejectedTasks(),
		m.GetTimeoutTasks(),
		m.GetCompletedTasks(),
		m.GetErrorCount(),
		m.GetActiveWorkers(),
		m.GetAverageTime(),
	)

	return base
}
