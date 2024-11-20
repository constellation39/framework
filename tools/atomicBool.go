package tools

import (
	"sync/atomic"
)

// Bool 是一个原子操作的布尔类型
type Bool struct {
	value uint32
}

// NewBool 创建一个新的原子布尔值
func NewBool(initial bool) *Bool {
	b := &Bool{}
	if initial {
		b.Set(true)
	}
	return b
}

// Get 原子地获取当前值
func (b *Bool) Get() bool {
	return atomic.LoadUint32(&b.value) != 0
}

// Set 原子地设置新值
func (b *Bool) Set(value bool) {
	if value {
		atomic.StoreUint32(&b.value, 1)
	} else {
		atomic.StoreUint32(&b.value, 0)
	}
}

// SetTo 原子地设置为指定值并返回之前的值
func (b *Bool) SetTo(value bool) bool {
	var newValue uint32
	if value {
		newValue = 1
	}
	return atomic.SwapUint32(&b.value, newValue) != 0
}

// CompareAndSwap 原子地比较并交换值
func (b *Bool) CompareAndSwap(old, new bool) bool {
	var oldValue, newValue uint32
	if old {
		oldValue = 1
	}
	if new {
		newValue = 1
	}
	return atomic.CompareAndSwapUint32(&b.value, oldValue, newValue)
}

// Toggle 原子地切换布尔值并返回新值
func (b *Bool) Toggle() bool {
	for {
		old := atomic.LoadUint32(&b.value)
		new := uint32(1)
		if old != 0 {
			new = 0
		}
		if atomic.CompareAndSwapUint32(&b.value, old, new) {
			return new != 0
		}
	}
}

// String 实现 Stringer 接口
func (b *Bool) String() string {
	if b.Get() {
		return "true"
	}
	return "false"
}
