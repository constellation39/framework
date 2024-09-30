package tools

import "sync/atomic"

type AtomicBool struct {
	value int32
}

func NewAtomicBool(initial bool) *AtomicBool {
	var intValue int32
	if initial {
		intValue = 1
	} else {
		intValue = 0
	}
	return &AtomicBool{
		value: intValue,
	}
}

func (ab *AtomicBool) IsTrue() bool {
	return atomic.LoadInt32(&ab.value) == 1
}
func (ab *AtomicBool) IsFalse() bool {
	return atomic.LoadInt32(&ab.value) == 0
}

func (ab *AtomicBool) Set(newValue bool) {
	var intValue int32
	if newValue {
		intValue = 1
	} else {
		intValue = 0
	}
	atomic.StoreInt32(&ab.value, intValue)
}
