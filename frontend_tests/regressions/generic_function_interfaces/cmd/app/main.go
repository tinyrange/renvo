package main

import "example.com/genericfunctioninterfaces/model"

type Container struct{ Value model.Callback[int] }

type Blank struct{ _ model.Callback[int] }

func comparisonPanics(a, b any) (caught bool) {
	defer func() {
		if recover() != nil {
			caught = true
		}
	}()
	_ = model.Equal[any](a, b)
	return
}

func mapKeyPanics(operation int, mapping map[any]int, key any) (caught bool) {
	defer func() {
		if recover() != nil {
			caught = true
		}
	}()
	switch operation {
	case 0:
		model.Put(mapping, key, 42)
	case 1:
		_ = model.Lookup(mapping, key)
	case 2:
		model.Delete(mapping, key)
	}
	return
}

func main() {
	f := model.New(2)
	boxed := model.Box(f)
	if !comparisonPanics(boxed, boxed) {
		panic("boxed function comparison")
	}
	var zero model.Callback[int]
	if model.Box(zero) == nil || !comparisonPanics(model.Box(zero), model.Box(zero)) {
		panic("typed nil function comparison")
	}
	if !comparisonPanics(model.Box([1]model.Callback[int]{f}), model.Box([1]model.Callback[int]{f})) {
		panic("function array comparison")
	}
	if !comparisonPanics(model.Box([0]model.Callback[int]{}), model.Box([0]model.Callback[int]{})) {
		panic("zero function array comparison")
	}
	if !comparisonPanics(model.Box(Container{Value: f}), model.Box(Container{Value: f})) {
		panic("function field comparison")
	}
	if !comparisonPanics(model.Box(Blank{}), model.Box(Blank{})) {
		panic("blank function field comparison")
	}
	if model.Equal[any](boxed, model.Box(42)) || model.Equal[any](boxed, nil) {
		panic("different dynamic types")
	}
	if !model.Equal[any](model.Box(&f), model.Box(&f)) || !model.Equal[any](model.Box(42), model.Box(42)) {
		panic("comparable values")
	}
	read, ok := model.Read[model.Alias[int]](boxed)
	if !ok || read == nil || zero != nil || read(40) != 42 || read.Apply(40) != 42 {
		panic("function identity and methods")
	}
	if _, ok := model.Read[model.Other[int]](boxed); ok {
		panic("distinct defined function types")
	}
	if _, ok := model.Read[func(int) int](boxed); ok {
		panic("defined function acquired anonymous identity")
	}
	var anonymous func(int) int = f
	if anonymous(40) != 42 {
		panic("assignment from defined function")
	}
	if _, ok := model.Read[model.Callback[int]](model.Box(anonymous)); ok {
		panic("anonymous function acquired defined identity")
	}
	unnamed, ok := model.Read[model.Unnamed[int]](model.Box(anonymous))
	if !ok || unnamed(40) != 42 {
		panic("anonymous function alias identity")
	}
	var converted model.Callback[int] = unnamed
	if converted(40) != 42 || model.ToUnnamed(f)(40) != 42 || model.ToDefined(unnamed)(40) != 42 {
		panic("function assignment and return dispatch")
	}
	other := model.NewOther(12)
	otherConverted := model.Callback[int](other)
	if otherConverted(40) != 52 || model.Other[int](f)(40) != 42 {
		panic("conversion between defined functions")
	}
	var otherAnonymous func(int) int = other
	if otherAnonymous(40) != 52 || anonymous(40) != 42 {
		panic("shared function tags")
	}
	for _, operation := range []int{0, 1, 2} {
		if !mapKeyPanics(operation, map[any]int{}, boxed) || !mapKeyPanics(operation, nil, model.Box(zero)) {
			panic("function map key")
		}
	}
	print("PASS\n")
}
