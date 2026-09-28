//go:build !renvo

package json

import "reflect"

// Host tests exercise our encoder with Go reflection. Renvo annotation policy
// is tested in native fixtures, since the host toolchain cannot observe it.
func scalarValue(value any) (any, bool) {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil, false
	}
	switch v.Kind() {
	case reflect.String:
		return v.String(), true
	case reflect.Bool:
		return v.Bool(), true
	case reflect.Int:
		return int(v.Int()), true
	case reflect.Int8:
		return int8(v.Int()), true
	case reflect.Int16:
		return int16(v.Int()), true
	case reflect.Int32:
		return int32(v.Int()), true
	case reflect.Int64:
		return v.Int(), true
	case reflect.Uint:
		return uint(v.Uint()), true
	case reflect.Uint8:
		return uint8(v.Uint()), true
	case reflect.Uint16:
		return uint16(v.Uint()), true
	case reflect.Uint32:
		return uint32(v.Uint()), true
	case reflect.Uint64:
		return v.Uint(), true
	case reflect.Uintptr:
		return uintptr(v.Uint()), true
	case reflect.Float32:
		return float32(v.Float()), true
	case reflect.Float64:
		return v.Float(), true
	}
	return nil, false
}
func describeFields(value any) ([]fieldInfo, bool) {
	typ := reflect.TypeOf(value)
	if typ == nil {
		return nil, false
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, false
	}
	var fields []fieldInfo
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			fields = append(fields, fieldInfo{Name: field.Name, Tag: string(field.Tag)})
		}
	}
	return fields, true
}
func readField(value any, index int) (any, bool) {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil, false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, false
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).IsExported() {
			if index == 0 {
				return v.Field(i).Interface(), true
			}
			index--
		}
	}
	return nil, false
}
func inspectCollection(value any) (collectionInfo, bool) {
	v := reflect.ValueOf(value)
	var out collectionInfo
	if !v.IsValid() {
		return out, false
	}
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
		out.Key = reflect.Zero(v.Type().Key()).Interface()
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

func copyStruct(value any) (any, bool) {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil, false
	}
	typ := v.Type()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, false
	}
	out := reflect.New(typ)
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			out.Elem().Set(v.Elem())
		}
	} else {
		out.Elem().Set(v)
	}
	return out.Interface(), true
}

func hostAssign(target reflect.Value, value any) bool {
	if !target.CanSet() {
		return false
	}
	v := reflect.ValueOf(value)
	if target.Kind() == reflect.Interface && target.NumMethod() == 0 {
		if !v.IsValid() {
			target.SetZero()
		} else {
			target.Set(v)
		}
		return true
	}
	if !v.IsValid() || v.Type() != target.Type() {
		return false
	}
	target.Set(v)
	return true
}

func assignValue(target, value any) bool {
	v := reflect.ValueOf(target)
	return v.IsValid() && v.Kind() == reflect.Pointer && !v.IsNil() && hostAssign(v.Elem(), value)
}

func writeField(target any, index int, value any) bool {
	v := reflect.ValueOf(target)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return false
	}
	v = v.Elem()
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).IsExported() {
			if index == 0 {
				return hostAssign(v.Field(i), value)
			}
			index--
		}
	}
	return false
}

func scalarLike(prototype, value any) (any, bool) {
	typ := reflect.TypeOf(prototype)
	v := reflect.ValueOf(value)
	if typ == nil || !v.IsValid() || typ.Kind() != v.Kind() || !v.Type().ConvertibleTo(typ) {
		return nil, false
	}
	return v.Convert(typ).Interface(), true
}

func rebuildCollection(prototype any, keys, values []any, nilValue bool) (any, bool) {
	typ := reflect.TypeOf(prototype)
	if typ == nil {
		return nil, false
	}
	kind := typ.Kind()
	if kind != reflect.Pointer && kind != reflect.Slice && kind != reflect.Array && kind != reflect.Map {
		return nil, false
	}
	if nilValue {
		if kind == reflect.Array || len(keys) != 0 || len(values) != 0 {
			return nil, false
		}
		return reflect.Zero(typ).Interface(), true
	}
	if kind != reflect.Map && len(keys) != 0 {
		return nil, false
	}
	var out reflect.Value
	switch kind {
	case reflect.Pointer:
		if len(values) != 1 {
			return nil, false
		}
		out = reflect.New(typ.Elem())
		if !hostAssign(out.Elem(), values[0]) {
			return nil, false
		}
		return out.Convert(typ).Interface(), true
	case reflect.Slice:
		out = reflect.MakeSlice(typ, len(values), len(values))
	case reflect.Array:
		if len(values) != typ.Len() {
			return nil, false
		}
		out = reflect.New(typ).Elem()
	case reflect.Map:
		if len(keys) != len(values) {
			return nil, false
		}
		out = reflect.MakeMap(typ)
	}
	for i, value := range values {
		if kind == reflect.Map {
			key := reflect.New(typ.Key()).Elem()
			item := reflect.New(typ.Elem()).Elem()
			if !hostAssign(key, keys[i]) || !hostAssign(item, value) {
				return nil, false
			}
			out.SetMapIndex(key, item)
		} else if !hostAssign(out.Index(i), value) {
			return nil, false
		}
	}
	return out.Interface(), true
}
