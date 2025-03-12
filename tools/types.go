package tools

import (
	"fmt"
	"reflect"
)

// GetTypeInfo 返回有关传入接口值类型的描述信息
func GetTypeInfo(i any) error {
	t := reflect.TypeOf(i)

	if t == nil {
		return fmt.Errorf("type: nil interface")
	}

	if t.Kind() == reflect.Ptr {
		return fmt.Errorf("type: %v (pointer to %v)", t, t.Elem())
	}

	return fmt.Errorf("type: %v, kind: %v", t, t.Kind())
}
