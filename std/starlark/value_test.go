package starlark

import (
	"strings"
	"testing"
)

func TestValueCollectionsAndStructs(t *testing.T) {
	original := []Value{Int(1), String("two")}
	list := NewList(original)
	original[0] = Int(9)
	if list.String() != `[1, "two"]` {
		t.Fatalf("NewList did not copy its input: %s", list)
	}

	dict := NewDict()
	if err := dict.SetKey(String("b"), Int(2)); err != nil {
		t.Fatal(err)
	}
	if err := dict.SetKey(String("a"), Int(1)); err != nil {
		t.Fatal(err)
	}
	if err := dict.SetKey(String("b"), Int(3)); err != nil {
		t.Fatal(err)
	}
	value, found, err := dict.Get(String("b"))
	if err != nil || !found || value != Int(3) || dict.Len() != 2 {
		t.Fatalf("dict lookup = (%v, %v, %v), len=%d", value, found, err, dict.Len())
	}
	if err := dict.SetKey(NewList(nil), None); err == nil || !strings.Contains(err.Error(), "unhashable") {
		t.Fatalf("unhashable key error = %v", err)
	}

	fields := StringDict{"z": Int(3), "a": String("first")}
	object := NewStruct(String("record"), fields)
	fields["a"] = String("changed")
	if object.Type() != "record" || object.String() != `struct(a = "first", z = 3)` {
		t.Fatalf("struct = %s (%s)", object, object.Type())
	}
	if names := object.AttrNames(); len(names) != 2 || names[0] != "a" || names[1] != "z" {
		t.Fatalf("attribute names = %v", names)
	}
	if value, err := object.Attr("missing"); err != nil || value != nil {
		t.Fatalf("missing attribute = (%v, %v)", value, err)
	}
	if keys := (StringDict{"z": None, "a": None}).Keys(); len(keys) != 2 || keys[0] != "a" || keys[1] != "z" {
		t.Fatalf("sorted keys = %v", keys)
	}
}

func TestIteratorsAndStructuralEquality(t *testing.T) {
	iter := Tuple{Int(1), String("two")}.Iterate()
	var values []string
	var value Value
	for iter.Next(&value) {
		values = append(values, value.String())
	}
	iter.Done()
	if strings.Join(values, ",") != `1,"two"` {
		t.Fatalf("iteration = %v", values)
	}

	left := NewList([]Value{Int(1), Tuple{String("x")}})
	right := NewList([]Value{Int(1), Tuple{String("x")}})
	if !equal(left, right) {
		t.Fatal("equal nested lists compare unequal")
	}
	first := NewDict()
	second := NewDict()
	_ = first.SetKey(String("one"), left)
	_ = second.SetKey(String("one"), right)
	if !equal(first, second) {
		t.Fatal("equal dictionaries compare unequal")
	}
}

func TestUnpackArgs(t *testing.T) {
	var name string
	count := 7
	enabled := false
	var value Value
	err := UnpackArgs("demo",
		Tuple{String("renvo")},
		[]Tuple{{String("enabled"), True}, {String("value"), Int(9)}},
		"name", &name,
		"count?", &count,
		"enabled", &enabled,
		"value", &value,
	)
	if err != nil || name != "renvo" || count != 7 || !enabled || value != Int(9) {
		t.Fatalf("UnpackArgs = name=%q count=%d enabled=%v value=%v err=%v", name, count, enabled, value, err)
	}

	var number int
	if err := UnpackArgs("demo", Tuple{String("bad")}, nil, "number", &number); err == nil || !strings.Contains(err.Error(), "want int") {
		t.Fatalf("type error = %v", err)
	}
	if err := UnpackArgs("demo", Tuple{Int(1)}, []Tuple{{String("number"), Int(2)}}, "number", &number); err == nil || !strings.Contains(err.Error(), "multiple values") {
		t.Fatalf("duplicate error = %v", err)
	}
	if err := UnpackArgs("demo", nil, nil, 1, &number); err == nil || !strings.Contains(err.Error(), "invalid host signature") {
		t.Fatalf("host signature error = %v", err)
	}
}

func TestThreadAndBuiltinHostAPI(t *testing.T) {
	thread := &Thread{Name: "host"}
	thread.SetLocal("request", "value")
	if got := thread.Local("request"); got != "value" {
		t.Fatalf("thread local = %v", got)
	}

	called := false
	builtin := NewBuiltin("inspect", func(gotThread *Thread, gotBuiltin *Builtin, args Tuple, kwargs []Tuple) (Value, error) {
		called = gotThread == thread && gotBuiltin.Name() == "inspect" && len(args) == 1 && len(kwargs) == 1
		return args[0], nil
	})
	result, err := Call(thread, builtin, Tuple{Int(42)}, []Tuple{{String("flag"), True}})
	if err != nil || !called || result != Int(42) {
		t.Fatalf("Call = (%v, %v), called=%v", result, err, called)
	}

	thread.Cancel("test requested")
	if _, err := ExecFile(thread, "cancel.star", "value = 1", nil); err == nil || !strings.Contains(err.Error(), "test requested") {
		t.Fatalf("cancel error = %v", err)
	}
}
