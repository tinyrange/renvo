//go:build renvo

package json

import "reflect"

func scalarValue(value any) (any, bool) { return reflect.Scalar(value) }
func describeFields(value any) ([]fieldInfo, bool) {
	meta, ok := reflect.Describe(value)
	var fields []fieldInfo
	for _, field := range meta.Fields {
		fields = append(fields, fieldInfo{Name: field.Name, Tag: field.Tag})
	}
	return fields, ok
}
func readField(value any, index int) (any, bool) { return reflect.FieldValue(value, index) }
func inspectCollection(value any) (collectionInfo, bool) {
	meta, ok := reflect.InspectCollection(value)
	return collectionInfo{Kind: meta.Kind, Nil: meta.Nil, Keys: meta.Keys, Values: meta.Values, Element: meta.Element, Key: meta.Key}, ok
}

func copyStruct(value any) (any, bool)                 { return reflect.StructCopy(value) }
func assignValue(target, value any) bool               { return reflect.Assign(target, value) }
func writeField(target any, index int, value any) bool { return reflect.SetField(target, index, value) }
func scalarLike(prototype, value any) (any, bool)      { return reflect.ScalarLike(prototype, value) }
func rebuildCollection(prototype any, keys, values []any, nilValue bool) (any, bool) {
	return reflect.RebuildCollection(prototype, keys, values, nilValue)
}
