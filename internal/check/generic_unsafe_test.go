package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericUnsafeLayoutOperands(t *testing.T) {
	for _, test := range []struct {
		name, declarations, body string
		valid                    bool
	}{
		{"parameter", "", `var v T;_=unsafe.Sizeof(v);_=unsafe.Alignof(v)`, true},
		{"constant_scalar", "", `const n=unsafe.Sizeof(uint64(0));var v [n]T;_=v`, true},
		{"constant_scalar_variable", "", `var v int;const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_zero_struct", "", `const n=unsafe.Sizeof(struct{}{});var a [n]T;var b [0]T=a;_=b`, true},
		{"constant_zero_array", "", `const n=unsafe.Sizeof([0]int{});var a [n]T;var b [0]T=a;_=b`, true},
		{"constant_zero_struct_array", "", `const n=unsafe.Sizeof([10]struct{}{});var a [n]T;var b [0]T=a;_=b`, true},
		{"constant_parameter", "", `var v T;const n=unsafe.Sizeof(v);_=n`, false},
		{"constant_parameter_array", "", `var v [0]T;const n=unsafe.Sizeof(v);_=n`, false},
		{"constant_pointer", "", `var v *T;const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_slice", "", `var v []T;const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_map", "", `var v map[int]T;const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_channel", "", `var v chan T;const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_interface", "", `var v any;const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_struct", "", `var v struct{Value []T};const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_fixed_offset", "", `var v struct{Value []T;Last int};const n=unsafe.Offsetof(v.Last);var a [n]T;_=a`, true},
		{"constant_pointer_alignment", "", `var v *T;const n=unsafe.Alignof(v);var a [n]T;_=a`, true},
		{"constant_function_alignment", "", `var v func(T);const n=unsafe.Alignof(v);var a [n]T;_=a`, true},
		{"constant_function_size", "", `var v func(T);const n=unsafe.Sizeof(v);var a [n]T;_=a`, true},
		{"constant_function_field", "", `var v struct{Callback func(T);Last int};const n=unsafe.Offsetof(v.Last);var a [n]T;_=a`, true},
		{"typed_nil", "", `_=unsafe.Sizeof((*T)(nil));_=unsafe.Alignof(([]T)(nil))`, true},
		{"default_values", "", `_=unsafe.Sizeof(1);_=unsafe.Sizeof(1.0);_=unsafe.Sizeof(1i);_=unsafe.Sizeof("value");_=unsafe.Alignof(true)`, true},
		{"untyped_nil_size", "", `_=unsafe.Sizeof(nil)`, false},
		{"untyped_nil_align", "", `_=unsafe.Alignof(nil)`, false},
		{"overflow_integer", "", `_=unsafe.Sizeof(1<<100)`, false},
		{"overflow_float", "", `_=unsafe.Alignof(1e1000)`, false},
		{"void", `func Void(){}`, `_=unsafe.Sizeof(Void())`, false},
		{"tuple", `func Pair()(int,int){return 1,2}`, `_=unsafe.Sizeof(Pair())`, false},
		{"type", `type Record struct{Value int}`, `_=unsafe.Sizeof(Record)`, false},
		{"parameter_type", "", `_=unsafe.Alignof(T)`, false},
		{"basic_type", "", `_=unsafe.Sizeof(int)`, false},
		{"function", `func Value()int{return 1}`, `_=unsafe.Sizeof(Value)`, true},
		{"generic_function", `func Value[U any](v U)U{return v}`, `_=unsafe.Sizeof(Value)`, false},
		{"instantiated_function", `func Value[U any](v U)U{return v}`, `_=unsafe.Sizeof(Value[int])`, true},
		{"missing_argument", "", `_=unsafe.Sizeof()`, false},
		{"extra_argument", "", `_=unsafe.Alignof(1,2)`, false},
		{"spread", "", `var values []int;_=unsafe.Sizeof(values...)`, false},
		{"selector", `type Record struct{Value int}`, `var v Record;_=unsafe.Offsetof(v.Value)`, true},
		{"parenthesized_selector", `type Record struct{Value int}`, `var v Record;_=unsafe.Offsetof(((v.Value)))`, true},
		{"pointer_base", `type Record struct{Value int}`, `var v *Record;_=unsafe.Offsetof(v.Value)`, true},
		{"explicit_pointer", `type Record struct{Value int};type Outer struct{*Record}`, `var v Outer;_=unsafe.Offsetof(v.Record.Value)`, true},
		{"value_promotion", `type Record struct{Value int};type Middle struct{Record};type Outer struct{Middle}`, `var v Outer;_=unsafe.Offsetof(v.Value)`, true},
		{"pointer_promotion", `type Record struct{Value int};type Outer struct{*Record}`, `var v Outer;_=unsafe.Offsetof(v.Value)`, false},
		{"pointer_promotion_deep", `type Record struct{Value int};type Middle struct{*Record};type Outer struct{Middle}`, `var v *Outer;_=unsafe.Offsetof(v.Value)`, false},
		{"ambiguous", `type Left struct{Value int};type Right struct{Value int};type Outer struct{Left;Right}`, `var v Outer;_=unsafe.Offsetof(v.Value)`, false},
		{"shallower", `type Left struct{Value int};type Middle struct{Left};type Right struct{Value int};type Outer struct{Middle;Right}`, `var v Outer;_=unsafe.Offsetof(v.Value)`, true},
		{"method", `type Record int;func(v Record)Value()int{return int(v)}`, `var v Record;_=unsafe.Offsetof(v.Value)`, false},
		{"method_blocks_field", `type Record int;func(v Record)Value()int{return int(v)};type Inner struct{Value int};type Middle struct{Inner};type Outer struct{Record;Middle}`, `var v Outer;_=unsafe.Offsetof(v.Value)`, false},
		{"missing_field", `type Record struct{Value int}`, `var v Record;_=unsafe.Offsetof(v.Absent)`, false},
		{"nonselector", "", `_=unsafe.Offsetof(1)`, false},
		{"parameter_field", "", `var v T;_=unsafe.Offsetof(v.Value)`, false},
		{"array_length", `func Value()int{return 1}`, `const n=len([3]int{int(unsafe.Sizeof(Value()))});var a [n]T;_=a`, true},
		{"array_length_variable_call", `func Value()int{return 1}`, `const n=len([3]int{Value()});var a [n]T;_=a`, false},
		{"array_length_cycle", `const N=len([N]int{});type Array[U any] [N]U`, `var a Array[T];_=a`, false},
		{"fixed_size_unevaluated_call", `func Value()int{return 1}`, `const n=len([2]int{int(unsafe.Sizeof(Value()))});_=n`, true},
		{"fixed_align_unevaluated_call", `func Value()int{return 1}`, `const n=cap([2]int{int(unsafe.Alignof(Value()))});_=n`, true},
		{"fixed_offset_unevaluated_call", `type Record struct{Value int};func Value()Record{return Record{1}}`, `const n=len([2]int{int(unsafe.Offsetof(Value().Value))});_=n`, true},
		{"variable_size_call", "", `var v T;const n=len([2]int{int(unsafe.Sizeof(v))});_=n`, false},
		{"variable_align_call", "", `var v [0]T;const n=len([2]int{int(unsafe.Alignof(v))});_=n`, false},
		{"fixed_indirection", "", `var v *T;const n=len([2]int{int(unsafe.Sizeof(v))});_=n`, true},
		{"fixed_slice", "", `var v []T;const n=len([2]int{int(unsafe.Alignof(v))});_=n`, true},
		{"variable_struct", "", `var v struct{Value T};const n=len([2]int{int(unsafe.Sizeof(v))});_=n`, false},
		{"fixed_struct", "", `var v struct{Value []T};const n=len([2]int{int(unsafe.Sizeof(v))});_=n`, true},
		{"variable_offset", `type Record[U any] struct{First int;Value U}`, `var v Record[T];const n=len([2]int{int(unsafe.Offsetof(v.First))});_=n`, false},
		{"fixed_offset", `type Record[U any] struct{Value []U;First int}`, `var v Record[T];const n=len([2]int{int(unsafe.Offsetof(v.First))});_=n`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;import \"unsafe\";" + test.declarations + ";func F[T any]() {" + test.body + "};func main(){}")},
				{Path: "/std/unsafe/unsafe.go", Src: []byte("package unsafe;type Pointer *byte")},
			})
			got := PrepareGenerics(graph)
			if got.Ok != test.valid {
				t.Fatalf("ok=%v want=%v: %s", got.Ok, test.valid, got.Message)
			}
		})
	}
}
