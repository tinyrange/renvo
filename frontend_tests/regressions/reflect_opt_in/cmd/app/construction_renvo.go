//go:build renvo

package main

import "reflect"

func checkConstruction() {
	original := Record{A: 7, hidden: "private"}
	copyValue, ok := reflect.StructCopy(original)
	if !ok {
		panic("copy metadata")
	}
	copied, ok := copyValue.(*Record)
	if !ok || copied.A != 7 || copied.hidden != "private" {
		panic("copy value")
	}
	copied.A = 9
	if original.A != 7 {
		panic("copy aliases original")
	}
	var target Record
	if !reflect.Assign(&target, *copied) || target.A != 9 || target.hidden != "private" {
		panic("assign struct")
	}
	if reflect.Assign(&target, Plain{}) || target.A != 9 {
		panic("assign mismatch")
	}
	var absent *Record
	zero, ok := reflect.StructCopy(absent)
	if !ok {
		panic("nil copy metadata")
	}
	z, ok := zero.(*Record)
	if !ok || z == nil || z.A != 0 {
		panic("nil copy value")
	}
	if reflect.Assign(absent, original) {
		panic("nil target accepted")
	}
	if _, ok := reflect.StructCopy(Plain{}); ok {
		panic("unannotated copy")
	}
	var dynamic any = "old"
	if !reflect.Assign(&dynamic, nil) { panic("assign nil interface returned false") }
	if dynamic != nil { panic("assign nil interface value") }
	if !reflect.Assign(&dynamic, 42) {
		panic("assign dynamic")
	}
	n, ok := dynamic.(int)
	if !ok || n != 42 {
		panic("assign dynamic result")
	}
	v, ok := reflect.RebuildCollection([]any{}, nil, []any{nil, "x"}, false)
	if !ok {
		panic("generic slice rebuild")
	}
	a, ok := v.([]any)
	if !ok || len(a) != 2 || a[0] != nil {
		panic("generic slice nil")
	}
	meta, ok := reflect.InspectCollection(map[string]any{})
	if !ok {
		panic("generic map metadata")
	}
	key, ok := meta.Key.(string)
	if !ok || key != "" {
		panic("typed zero map key")
	}
}
