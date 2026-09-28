//go:build renvo

package main

import "reflect"

const optInEnforced = true

func inspect(value any) (collection, bool) {
	v, ok := reflect.InspectCollection(value)
	return collection{Kind: v.Kind, Nil: v.Nil, Keys: v.Keys, Values: v.Values, Element: v.Element}, ok
}
func rebuild(value any, keys []any, values []any, nilValue bool) (any, bool) {
	return reflect.RebuildCollection(value, keys, values, nilValue)
}

func read(value any, index int) (any, bool) { return reflect.FieldValue(value, index) }
func write(value any, index int, replacement any) bool {
	return reflect.SetField(value, index, replacement)
}

func describe(value any) (string, []field, bool) {
	meta, ok := reflect.Describe(value)
	var fields []field
	for _, item := range meta.Fields {
		fields = append(fields, field{Name: item.Name, Tag: item.Tag})
	}
	return meta.Name, fields, ok
}
