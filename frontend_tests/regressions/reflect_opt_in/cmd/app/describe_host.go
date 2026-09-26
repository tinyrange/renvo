//go:build !renvo

package main

import "reflect"

// Host Go supplies a positive metadata oracle, but does not implement Renvo's
// opt-in policy. That policy is asserted only by the Renvo execution.
const optInEnforced = false

func inspect(value any) (collection, bool) {
	v := reflect.ValueOf(value)
	var out collection
	switch v.Kind() {
	case reflect.Slice:
		out.Kind = "slice"
		out.Nil = v.IsNil()
	case reflect.Array:
		out.Kind = "array"
	case reflect.Map:
		out.Kind = "map"
		out.Nil = v.IsNil()
	case reflect.Pointer:
		out.Kind = "pointer"
		out.Nil = v.IsNil()
	default:
		return out, false
	}
	out.Element = reflect.Zero(v.Type().Elem()).Interface()
	if out.Kind == "pointer" {
		if !out.Nil {
			out.Values = append(out.Values, v.Elem().Interface())
		}
	} else if out.Kind == "map" {
		iter := v.MapRange()
		for iter.Next() {
			out.Keys = append(out.Keys, iter.Key().Interface())
			out.Values = append(out.Values, iter.Value().Interface())
		}
	} else {
		for i := 0; i < v.Len(); i++ {
			out.Values = append(out.Values, v.Index(i).Interface())
		}
	}
	return out, true
}
func rebuild(value any, keys []any, values []any, nilValue bool) (any, bool) {
	typ := reflect.TypeOf(value)
	if typ == nil {
		return nil, false
	}
	kind := typ.Kind()
	if kind != reflect.Slice && kind != reflect.Array && kind != reflect.Map && kind != reflect.Pointer {
		return nil, false
	}
	if nilValue {
		if kind == reflect.Array || len(keys) != 0 || len(values) != 0 {
			return nil, false
		}
		return reflect.Zero(typ).Interface(), true
	}
	if kind == reflect.Map {
		if len(keys) != len(values) {
			return nil, false
		}
	} else if len(keys) != 0 {
		return nil, false
	}
	if kind == reflect.Array && typ.Len() != len(values) || kind == reflect.Pointer && len(values) != 1 {
		return nil, false
	}
	for i, item := range values {
		if reflect.TypeOf(item) != typ.Elem() {
			return nil, false
		}
		if kind == reflect.Map && reflect.TypeOf(keys[i]) != typ.Key() {
			return nil, false
		}
	}
	var out reflect.Value
	switch kind {
	case reflect.Slice:
		out = reflect.MakeSlice(typ, len(values), len(values))
	case reflect.Map:
		out = reflect.MakeMap(typ)
	case reflect.Array:
		out = reflect.New(typ).Elem()
	case reflect.Pointer:
		out = reflect.New(typ.Elem())
	}
	for i, item := range values {
		v := reflect.ValueOf(item)
		if kind == reflect.Map {
			out.SetMapIndex(reflect.ValueOf(keys[i]), v)
		} else if kind == reflect.Pointer {
			out.Elem().Set(v)
		} else {
			out.Index(i).Set(v)
		}
	}
	return out.Interface(), true
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
	v := selected(value, index)
	r := reflect.ValueOf(replacement)
	if !v.IsValid() || !v.CanSet() || !r.IsValid() || v.Type() != r.Type() {
		return false
	}
	v.Set(r)
	return true
}

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
