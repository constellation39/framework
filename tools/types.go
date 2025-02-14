package tools

import (
	"errors"
	"fmt"
	"reflect"
)

func GetInterfaceType(i any) error {
	t := reflect.TypeOf(i)

	if t == nil {
		return errors.New("interface is nil")
	}

	if t.Kind() == reflect.Ptr {
		return fmt.Errorf("type is pointer: %v, points to: %v", t, t.Elem())
	}

	return fmt.Errorf("type is: %v, kind is: %v", t, t.Kind())
}
