//go:build !renvo

package errors

import "reflect"

func asTarget(err error, target any) (bool, bool) {
	v := reflect.ValueOf(target)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() {
		return false, false
	}
	t := v.Type().Elem()
	if t.Kind() != reflect.Interface && !t.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return false, false
	}
	e := reflect.ValueOf(err)
	if !e.Type().AssignableTo(t) {
		return true, false
	}
	v.Elem().Set(e)
	return true, true
}
