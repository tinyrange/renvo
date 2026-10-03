package main

import "example.com/genericreflection/model"

type field struct{ Name, Tag string }
type Local int

func check(value any, want string) {
	name, fields, ok := describe(value)
	if !ok || name != want || len(fields) != 3 {
		panic("generic type descriptor")
	}
	if fields[0].Name != "Public" || fields[1].Name != "Alias" || fields[2].Name != "Item" {
		panic("embedded field names")
	}
	if fields[0].Tag != "" || fields[1].Tag != "" || fields[2].Tag != "json:\"item\"" {
		panic("generic field tags")
	}
}

func main() {
	box := model.New(42)
	check(box, "Box[int]")
	check(model.New("hello"), "Box[string]")
	check(model.New(model.Count(1)), "Box[example.com/genericreflection/model.Count]")
	check(model.New(model.CountAlias(1)), "Box[example.com/genericreflection/model.Count]")
	check(model.New(Local(1)), "Box[main.Local]")
	check(model.New(box), "Box[example.com/genericreflection/model.Box[int]]")
	check(model.New[any](nil), "Box[interface {}]")
	check(model.New[error](nil), "Box[error]")
	check(model.New[interface{ Error() string }](nil), "Box[interface { Error() string }]")
	check(model.New(struct{}{}), "Box[struct {}]")
	check(model.New(struct{ V int }{1}), "Box[struct { V int }]")
	var empty *model.Box[int]
	check(empty, "Box[int]")
	value, found := read(box, 0)
	public, valid := value.(model.Public[int])
	if !found || !valid || public.Value != 42 {
		panic("generic embedded read")
	}
	value, found = read(box, 1)
	alias, valid := value.(model.Alias[int])
	if !found || !valid || alias.Value != 42 {
		panic("generic alias read")
	}
	if !write(&box, 1, model.Public[int]{51}) || box.Alias.Value != 51 || box.Public.Value != 42 {
		panic("generic alias write")
	}
	if !write(&box, 2, 61) || box.Item != 61 || box.Hidden() != 42 {
		panic("private field visibility")
	}
	if write(&box, 2, "wrong") || write(box, 2, 71) || write(&box, 3, model.Public[int]{71}) || write(empty, 2, 71) {
		panic("invalid generic write")
	}
	_, found = read(empty, 0)
	if found {
		panic("nil generic read")
	}
	_, found = read(box, 3)
	if found {
		panic("private generic read")
	}
	name, fields, ok := describe(model.Grouped[string]{"group"})
	if !ok || name != "Grouped[string]" || len(fields) != 1 || fields[0].Name != "Value" {
		panic("grouped generic annotation")
	}
	if optInEnforced {
		_, _, ok = describe(model.Plain[int]{1})
		if ok {
			panic("generic reflection requires opt-in")
		}
		_, _, ok = describe(model.Public[int]{1})
		if ok {
			panic("embedded type implicitly opted in")
		}
	}
	print("PASS\n")
}
