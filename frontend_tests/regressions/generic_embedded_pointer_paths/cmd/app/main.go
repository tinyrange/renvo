package main

import "example.com/genericembeddedpaths/model"

type Text string

var global model.Extra[Text]

func failed(f func()) bool {
	result := false
	func() {
		defer func() { result = recover() != nil }()
		f()
	}()
	return result
}

func main() {
	leaf := model.Leaf[Text]{Padding: 11, Value: "first"}
	middle := model.Middle[Text]{Padding: 13, Leaf: &leaf}
	valueMiddle := model.ValueMiddle[Text]{Padding: "padding", Middle: middle}
	outer := model.Outer[Text]{Padding: 17, ValueMiddle: &valueMiddle}
	extra := model.Extra[Text]{Padding: 19, Outer: &outer}
	global = extra
	if model.Read(extra) != "first" || model.ReadPointer(&extra) != "first" || global.Value != "first" {
		panic("nested pointer reads")
	}
	model.Set(&extra, "second")
	if leaf.Value != "second" || *model.Address(&extra) != "second" {
		panic("nested pointer writes")
	}
	values := []model.Extra[Text]{extra}
	if model.ReadSlice(values, 0) != "second" || model.ReadCall(extra) != "second" || model.ReadPointerCall(&extra) != "second" {
		panic("selector bases")
	}
	if model.ReadAssertion[Text](extra) != "second" || model.ReadPointerAssertion[Text](&extra) != "second" {
		panic("assertion bases")
	}
	get := func() Text { return extra.Value }
	set := func(v Text) { extra.Value = v }
	set("third")
	if get() != "third" || global.Value != "third" {
		panic("captured and global bases")
	}
	if !failed(func() { var p *model.Extra[Text]; _ = model.ReadPointer(p) }) ||
		!failed(func() { var v model.Extra[Text]; _ = model.Read(v) }) {
		panic("outer nil pointer")
	}
	outer.ValueMiddle = nil
	if !failed(func() { _ = model.Read(extra) }) {
		panic("middle nil pointer")
	}
	outer.ValueMiddle = &valueMiddle
	valueMiddle.Leaf = nil
	if !failed(func() { model.Set(&extra, "unreachable") }) {
		panic("inner nil pointer")
	}
	shadow := model.Shadow[int]{Leaf: model.Leaf[int]{Value: 42}}
	if model.ReadMethod(shadow) != 77 || model.ReadMethod(&shadow) != 77 {
		panic("direct method shadowing")
	}
	bound := shadow.Get
	expression := model.Shadow[int].Get
	var view interface{ Get() int } = shadow
	if bound() != 77 || expression(shadow) != 77 || view.Get() != 77 {
		panic("method value shadowing")
	}
	count := 0
	for func() bool {
		for _, v := range []model.Shadow[int]{shadow} {
			if model.ReadMethod(v) != 77 {
				return false
			}
		}
		return true
	}() {
		count++
		break
	}
	if count != 1 {
		panic("closure loop header")
	}
	print("PASS\n")
}
