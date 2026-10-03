//go:build !renvo

package main

import "reflect"

const optInEnforced = false

func describe(value any) (string, []field, bool) {
	typ := reflect.TypeOf(value)
	if typ == nil {
		return "", nil, false
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return "", nil, false
	}
	var fields []field
	for i := 0; i < typ.NumField(); i++ {
		item := typ.Field(i)
		if item.IsExported() {
			fields = append(fields, field{Name: item.Name, Tag: string(item.Tag)})
		}
	}
	return typ.Name(), fields, true
}

func selected(value any, index int) reflect.Value {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return reflect.Value{}
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || index < 0 {
		return reflect.Value{}
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).IsExported() {
			if index == 0 {
				return v.Field(i)
			}
			index--
		}
	}
	return reflect.Value{}
}

func read(value any, index int) (any, bool) {
	v := selected(value, index)
	if !v.IsValid() {
		return nil, false
	}
	return v.Interface(), true
}

func write(value any, index int, replacement any) bool {
	v, r := selected(value, index), reflect.ValueOf(replacement)
	if !v.IsValid() || !v.CanSet() || !r.IsValid() || v.Type() != r.Type() {
		return false
	}
	v.Set(r)
	return true
}
