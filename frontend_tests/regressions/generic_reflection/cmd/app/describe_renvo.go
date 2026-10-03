//go:build renvo

package main

import "reflect"

const optInEnforced = true

func describe(value any) (string, []field, bool) {
	meta, ok := reflect.Describe(value)
	var fields []field
	for _, item := range meta.Fields {
		fields = append(fields, field{Name: item.Name, Tag: item.Tag})
	}
	return meta.Name, fields, ok
}

func read(value any, index int) (any, bool) { return reflect.FieldValue(value, index) }
func write(value any, index int, replacement any) bool {
	return reflect.SetField(value, index, replacement)
}
