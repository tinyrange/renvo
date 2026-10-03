package frontend_tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type frontendDiagnosticCase struct {
	name       string
	files      map[string]string
	wantCode   string
	wantFile   string
	wantDetail string
	wantLine   int
	wantColumn int
}

func TestFrontendStructuredDiagnostics(t *testing.T) {
	root := repoRoot(t)
	frontends := []struct {
		name   string
		config frontendConfig
	}{
		{name: "host", config: frontendCompiler(t, root)},
		{name: "stage3", config: selfHostedFrontendCompiler(t, root)},
	}

	cases := []frontendDiagnosticCase{
		{
			name:     "invalid_generics_array_parameter_hides_constant",
			files:    map[string]string{"cmd/app/main.go": "package main\nconst N=2\ntype Bad[N any] [N]int\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go", wantDetail: "type parameter used as a value",
		},
		{
			name:     "invalid_generics_local_array_type_hides_constant",
			files:    map[string]string{"cmd/app/main.go": "package main\nconst N=2\nfunc unused[T any](){type N int;var a [N]int;_=a}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go", wantDetail: "type used as a value",
		},
		{
			name:     "invalid_generics_ordinary_binary",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc Id[T any](v T)T{return v}\nfunc unused(){var a string;_=a+1}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "invalid operation for operand types", wantLine: 3, wantColumn: 31,
		},
		{
			name:     "invalid_generics_ordinary_named_binary",
			files:    map[string]string{"cmd/app/main.go": "package main\nimport \"example.com/diagnostic/model\"\nfunc unused(a model.A,b model.B){_=a+b}\nfunc main(){}\n", "model/model.go": "package model\ntype A int\ntype B int\nfunc Id[T any](v T)T{return v}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "invalid operation for operand types",
		},
		{
			name:     "invalid_generics_ordinary_missing_return",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc Id[T any](v T)T{return v}\nfunc unused()int{}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "missing return in function",
		},
		{
			name:     "invalid_generics_ordinary_index_constant",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc Id[T any](v T)T{return v}\nfunc unused(){const n=\"abc\"[0];_=n}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constant initializer must be a constant expression",
		},
		{
			name:     "invalid_generics_closure_for_condition",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](){for func()int{for range []int{1}{};return 1}(){}}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "for condition must be boolean",
		},
		{
			name:     "invalid_generics_closure_inner_range",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](){for func()bool{for range true{};return true}(){}}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "range is not valid for every type in the constraint",
		},
		{
			name:     "invalid_generics_shadowed_package_type",
			files:    map[string]string{"cmd/app/main.go": "package main\nvar int=0\nfunc unused[T any](v int){}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "name does not denote a type",
		},
		{
			name:     "invalid_generics_shadowed_any_constraint",
			files:    map[string]string{"cmd/app/main.go": "package main\nvar any=0\nfunc unused[T any](){}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "name does not denote a type",
		},
		{
			name:     "invalid_generics_shadowed_comparable_constraint",
			files:    map[string]string{"cmd/app/main.go": "package main\nvar comparable=0\nfunc unused[T comparable](){}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "name does not denote a type",
		},
		{
			name:     "invalid_generics_shadowed_local_argument",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](){}\nfunc main(){int:=0;unused[[]int]();_=int}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "name does not denote a type",
		},
		{
			name:     "invalid_generics_constraint_conversion_unused",
			files:    map[string]string{"cmd/app/main.go": "package main\ntype C interface{~int}\nfunc unused(){_=C(1)}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:     "invalid_generics_constraint_conversion_imported",
			files:    map[string]string{"cmd/app/main.go": "package main\nimport \"example.com/diagnostic/model\"\nfunc unused(){_=model.C(1)}\nfunc main(){}\n", "model/model.go": "package model\ntype C interface{~int}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:     "invalid_generics_constraint_value_closure",
			files:    map[string]string{"cmd/app/main.go": "package main\ntype C interface{~int}\nfunc unused(){_=func(){_=new(C)}}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:     "invalid_generics_constraint_alias_conversion",
			files:    map[string]string{"cmd/app/main.go": "package main\ntype C[T any]=interface{~int}\nfunc unused(){_=C[int](1)}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:     "invalid_generics_constraint_value_initializer",
			files:    map[string]string{"cmd/app/main.go": "package main\ntype C interface{~int}\nvar value=[]C{}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:     "invalid_generics_constraint_comparable_conversion",
			files:    map[string]string{"cmd/app/main.go": "package main\nfunc unused(){_=comparable(1)}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:     "invalid_generics_unsafe_nil_operand",
			files:    map[string]string{"cmd/app/main.go": "package main\nimport \"unsafe\"\nfunc unused[T any](){_=unsafe.Sizeof(nil)}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "unsafe layout operand must be a single value with a type",
		},
		{
			name:     "invalid_generics_unsafe_pointer_promotion",
			files:    map[string]string{"cmd/app/main.go": "package main\nimport \"unsafe\"\ntype Inner struct{Value int}\ntype Outer struct{*Inner}\nfunc unused[T any](){var value Outer;_=unsafe.Offsetof(value.Value)}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "unsafe.Offsetof field is embedded through a pointer",
		},
		{
			name:     "invalid_generics_unsafe_default_overflow",
			files:    map[string]string{"cmd/app/main.go": "package main\nimport \"unsafe\"\nfunc unused[T any](){_=unsafe.Alignof(1<<100)}\nfunc main(){}\n"},
			wantCode: "RENVO-CHECK-042", wantFile: "cmd/app/main.go",
			wantDetail: "unsafe layout operand overflows its default type",
		},
		{
			name:       "invalid_generics_unsafe_union_conversion",
			files:      map[string]string{"cmd/app/main.go": "package main\nimport \"unsafe\"\nfunc unused[P ~unsafe.Pointer|~uintptr](p P)uintptr{return uintptr(p)}\nfunc main(){}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "conversion is not valid for every type in the constraint",
		},
		{
			name:       "invalid_generics_unused_type_set",
			files:      map[string]string{"cmd/app/main.go": "package main\ntype C interface{int|~int}\nfunc main(){}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid or overlapping constraint terms",
		},
		{
			name:       "invalid_generics_constraint_value_without_instantiation",
			files:      map[string]string{"cmd/app/main.go": "package main\ntype C interface{~int}\nvar value C\nfunc main(){}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "constraint interface cannot be used as a value type",
		},
		{
			name:       "invalid_generics_constant_len_call",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc value()int{return 0}\nfunc unused[T any](){const n=len([1]int{value()});_=n}\nfunc main(){}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "constant initializer must be a constant expression",
		},
		{
			name:       "c_vla",
			files:      map[string]string{"cmd/app/main.c": "int inspect(int count) { int values[count]; return 0; }\nint main(void) { return 0; }\n"},
			wantCode:   "RENVO-C11-002",
			wantFile:   "cmd/app/main.c",
			wantDetail: "variable length arrays are not supported",
		},
		{
			name:       "syntax",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main( {\n"},
			wantCode:   "RENVO-PARSE-006",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid function or method declaration",
			wantLine:   3,
			wantColumn: 1,
		},
		{
			name:       "composite_literal_in_type_assertion",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var sensor any; _, _ = sensor.([]byte{0x01}) }\n"},
			wantCode:   "RENVO-CHECK-033",
			wantFile:   "cmd/app/main.go",
			wantDetail: "type assertion requires a type; found a composite literal",
			wantLine:   3,
			wantColumn: 52,
		},
		{
			name:       "package_clause",
			files:      map[string]string{"cmd/app/main.go": "package\n\nfunc main() {}\n"},
			wantCode:   "RENVO-PARSE-003",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid or missing package clause",
			wantLine:   1,
			wantColumn: 1,
		},
		{
			name:       "import_declaration",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport\nfunc main() {}\n"},
			wantCode:   "RENVO-PARSE-004",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid import declaration",
			wantLine:   3,
			wantColumn: 1,
		},
		{
			name:       "top_level_declaration",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nvar\nfunc main() {}\n"},
			wantCode:   "RENVO-PARSE-005",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid top-level declaration",
			wantLine:   3,
			wantColumn: 1,
		},
		{
			name:       "package_scope_statement",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nprint(1)\nfunc main() {}\n"},
			wantCode:   "RENVO-PARSE-007",
			wantFile:   "cmd/app/main.go",
			wantDetail: "unexpected statement or expression at package scope",
			wantLine:   3,
			wantColumn: 1,
		},
		{
			name:       "invalid_generics_declaration",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc identity[T any](value T) T { return value + value }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "operation is not valid for every type in the constraint",
		},
		{
			name:       "invalid_generics_binary_operand",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any]() { _ = int8(1) - 128 }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "operation is not valid for every type in the constraint",
		},
		{
			name:       "invalid_generics_array_union_index",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T ~[2]int|~[3]int](v T) { _ = v[2] }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "index exceeds length",
		},
		{
			name:       "invalid_generics_slice_bounds",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[S ~[]E,E any](s S,n int) { _ = s[2:n:1] }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "slice bounds are out of order or exceed length",
		},
		{
			name:       "invalid_generics_make_capacity",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[S ~[]E,E any]() { _ = make(S,2,1) }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "make length exceeds capacity",
		},
		{
			name:       "invalid_generics_shift_context",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](n uint) { _ = 1.0<<n }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "value cannot use its default type",
		},
		{
			name:       "invalid_generics_short_reuse",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](v T) { v,x:=1,2;_,_=v,x }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment is not valid for every type in the constraint",
		},
		{
			name:       "invalid_generics_literal_index",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](v T) { _ = [2]T{2:v} }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "literal index must be a constant within bounds",
		},
		{
			name:       "invalid_generics_duplicate_index",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any](v T) { _ = []T{1:v,0:v,v} }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "duplicate literal index",
		},
		{
			name:       "invalid_generics_function_address",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc unused[T any]() { _ = &func(){} }\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "address operand is not addressable",
		},
		{
			name:       "invalid_generics_instantiation",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc identity[T ~int](value T) T { return value }\nfunc main() { _ = identity[string](\"bad\") }\n"},
			wantCode:   "RENVO-CHECK-042",
			wantFile:   "cmd/app/main.go",
			wantDetail: "type argument does not satisfy constraint",
		},
		{
			name:       "unresolved_import",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"github.com/example/missing\"\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "unresolved import github.com/example/missing",
		},
		{
			name:       "c_import_requires_c_source",
			files:      map[string]string{"cmd/app/main.go": "package main\n\n/* #include <stdlib.h> */\nimport \"C\"\n\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-003",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid import",
		},
		{
			name:       "cgo_build_tag_is_enabled",
			files:      map[string]string{"cmd/app/main.go": "//go:build cgo\n\npackage main\n\nfunc main( {\n"},
			wantCode:   "RENVO-PARSE-006",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid function or method declaration",
		},
		{
			name: "c_direct_go_reference",
			files: map[string]string{
				"cmd/app/main.go": "package main\nimport \"C\"\nfunc main() { _ = c_value() }\n",
				"cmd/app/value.c": "int c_value(void) { return 42; }\n",
			},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/main.go",
			wantDetail: "undefined identifier",
		},
		{
			name: "c_unexported_go_reference",
			files: map[string]string{
				"cmd/app/main.go": "package main\n/* int call_go(void); */\nimport \"C\"\nfunc callback() int { return 42 }\nfunc main() { _ = C.call_go() }\n",
				"cmd/app/value.c": "extern int callback(void);\nint call_go(void) { return callback(); }\n",
			},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/value.c",
			wantDetail: "undefined identifier",
		},
		{
			name: "c_undeclared_selector",
			files: map[string]string{
				"cmd/app/main.go": "package main\nimport \"C\"\nfunc main() { _ = C.present() }\n",
				"cmd/app/value.c": "int present(void) { return 42; }\n",
			},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/main.go",
			wantDetail: "undefined identifier",
		},
		{
			name: "c_declaration_is_file_local",
			files: map[string]string{
				"cmd/app/main.go":  "package main\n/* int present(void); */\nimport \"C\"\nfunc main() {}\n",
				"cmd/app/other.go": "package main\nimport \"C\"\nfunc other() { _ = C.present() }\n",
				"cmd/app/value.c":  "int present(void) { return 42; }\n",
			},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/other.go",
			wantDetail: "undefined identifier",
		},
		{
			name:       "unavailable_standard_package",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"time\"\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-020",
			wantFile:   "cmd/app/main.go",
			wantDetail: "standard library package time is not included in this RENVO build",
		},
		{
			name:       "embed_pattern",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"embed\"\n\n//go:embed missing.txt\nvar value string\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-018",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid go:embed directive or pattern: missing.txt",
		},
		{
			name: "import_cycle",
			files: map[string]string{
				"cmd/app/main.go": "package main\n\nimport _ \"example.com/diagnostic/lib\"\n\nfunc main() {}\n",
				"lib/lib.go":      "package lib\n\nimport _ \"example.com/diagnostic/cmd/app\"\n",
			},
			wantCode:   "RENVO-LOAD-011",
			wantFile:   "lib/lib.go",
			wantDetail: "import cycle detected",
		},
		{
			name: "duplicate_declaration",
			files: map[string]string{
				"cmd/app/a.go": "package main\nvar collision int\n",
				"cmd/app/b.go": "package main\nfunc collision() {}\nfunc main() {}\n",
			},
			wantCode:   "RENVO-CHECK-002",
			wantFile:   "cmd/app/b.go",
			wantDetail: "duplicate declaration",
			wantLine:   2,
			wantColumn: 6,
		},
		{
			name:       "duplicate_parameter",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc value(item int, item string) {}\nfunc main() {}\n"},
			wantCode:   "RENVO-CHECK-006",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid name or scope",
			wantLine:   2,
			wantColumn: 22,
		},
		{
			name:       "return_count",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc value() int { return }\nfunc main() { _ = value() }\n"},
			wantCode:   "RENVO-CHECK-007",
			wantFile:   "cmd/app/main.go",
			wantDetail: "return value count does not match function results",
		},
		{
			name:       "assignment_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value int; value = \"wrong\"; _ = value }\n"},
			wantCode:   "RENVO-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "assignment_type_bool_from_int",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value bool; value = 1; _ = value }\n"},
			wantCode:   "RENVO-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "undefined_identifier",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { print(missing) }\n"},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/main.go",
			wantDetail: "undefined identifier",
		},
		{
			name:       "undefined_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nvar value Missing\nfunc main() { _ = value }\n"},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/main.go",
			wantDetail: "undefined identifier",
		},
		{
			name:       "invalid_inferred_assignment",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value int; other := \"bad\"; value = other; _ = value }\n"},
			wantCode:   "RENVO-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "invalid_operands",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { value := 1 + true; _ = value }\n"},
			wantCode:   "RENVO-CHECK-030",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid operation for operand types",
		},
		{
			name:       "invalid_call_argument_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc consume(value int) {}\nfunc main() { consume(\"bad\") }\n"},
			wantCode:   "RENVO-CHECK-016",
			wantFile:   "cmd/app/main.go",
			wantDetail: "call argument is not assignable to its parameter",
		},
		{
			name: "invalid_imported_call_arity",
			files: map[string]string{
				"cmd/app/main.go":  "package main\n\nimport \"example.com/diagnostic/helper\"\n\nfunc main() { helper.Needs() }\n",
				"helper/helper.go": "package helper\n\nfunc Needs(value string) int { return len(value) }\n",
			},
			wantCode:   "RENVO-CHECK-032",
			wantFile:   "cmd/app/main.go",
			wantDetail: "function call argument count does not match parameters",
		},
		{
			name:       "invalid_local_call_arity",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc needs(value string) int { return len(value) }\nfunc main() { needs() }\n"},
			wantCode:   "RENVO-CHECK-032",
			wantFile:   "cmd/app/main.go",
			wantDetail: "function call argument count does not match parameters",
		},
		{
			name:       "invalid_return_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc value() int { return \"bad\" }\nfunc main() { _ = value() }\n"},
			wantCode:   "RENVO-CHECK-031",
			wantFile:   "cmd/app/main.go",
			wantDetail: "return value is not assignable to the function result",
		},
		{
			name:       "unterminated_string",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { print(\"unterminated\n) }\n"},
			wantCode:   "RENVO-PARSE-001",
			wantFile:   "cmd/app/main.go",
			wantDetail: "source contains an invalid or unterminated token",
		},
		{
			name:       "invalid_goroutine",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { go len(\"value\") }\n"},
			wantCode:   "RENVO-CHECK-017",
			wantFile:   "cmd/app/main.go",
			wantDetail: "go statement requires a function call",
		},
		{
			name:       "invalid_channel_direction",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nvar values chan<- int\nfunc main() { _ = <-values }\n"},
			wantCode:   "RENVO-CHECK-018",
			wantFile:   "cmd/app/main.go",
			wantDetail: "channel direction does not permit this operation",
		},
		{
			name:       "invalid_select",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { select { default: default: } }\n"},
			wantCode:   "RENVO-CHECK-019",
			wantFile:   "cmd/app/main.go",
			wantDetail: "select has an invalid communication clause or multiple defaults",
		},
		{
			name: "unused_import",
			files: map[string]string{
				"cmd/app/main.go": "package main\n\nimport \"example.com/diagnostic/lib\"\n\nfunc main() {}\n",
				"lib/lib.go":      "package lib\n",
			},
			wantCode:   "RENVO-CHECK-010",
			wantFile:   "cmd/app/main.go",
			wantDetail: "import is not used",
		},
		{
			name:       "unused_local",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc main() { unused := 1 }\n"},
			wantCode:   "RENVO-CHECK-020",
			wantFile:   "cmd/app/main.go",
			wantDetail: "local variable is declared but not used",
			wantLine:   2,
			wantColumn: 15,
		},
		{
			name:       "missing_main",
			files:      map[string]string{"cmd/app/main.go": "package main\nvar value = 1\n"},
			wantCode:   "RENVO-CHECK-021",
			wantFile:   "cmd/app/main.go",
			wantDetail: "package main has no top-level func main()",
			wantLine:   1,
			wantColumn: 9,
		},
		{
			name:       "invalid_main_signature",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc main(value int) {}\n"},
			wantCode:   "RENVO-CHECK-022",
			wantFile:   "cmd/app/main.go",
			wantDetail: "func main must have no parameters or results",
			wantLine:   2,
			wantColumn: 6,
		},
		{
			name:       "main_method_is_not_entrypoint",
			files:      map[string]string{"cmd/app/main.go": "package main\ntype runner int\nfunc (runner) main() {}\n"},
			wantCode:   "RENVO-CHECK-023",
			wantFile:   "cmd/app/main.go",
			wantDetail: "method main does not define the package entry point",
			wantLine:   3,
			wantColumn: 15,
		},
		{
			name:       "unaddressable_array_slice",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc values() [3]int { return [3]int{} }\nfunc main() { _ = values()[:] }\n"},
			wantCode:   "RENVO-CHECK-024",
			wantFile:   "cmd/app/main.go",
			wantDetail: "cannot slice an unaddressable array value",
			wantLine:   3,
			wantColumn: 27,
		},
		{
			name:       "constant_array_index",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc main() { _ = [3]int{}[3] }\n"},
			wantCode:   "RENVO-CHECK-025",
			wantFile:   "cmd/app/main.go",
			wantDetail: "constant array index is out of bounds",
			wantLine:   2,
			wantColumn: 28,
		},
		{
			name:       "deferred_value_builtin",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc main() { defer len(\"value\") }\n"},
			wantCode:   "RENVO-CHECK-026",
			wantFile:   "cmd/app/main.go",
			wantDetail: "deferred builtin call discards a result",
			wantLine:   2,
			wantColumn: 21,
		},
		{
			name:       "builtin_arity",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc main() { _ = len() }\n"},
			wantCode:   "RENVO-CHECK-027",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid number of arguments to builtin",
			wantLine:   2,
			wantColumn: 19,
		},
		{
			name:       "builtin_operand",
			files:      map[string]string{"cmd/app/main.go": "package main\nfunc main() { _ = min(1, \"value\") }\n"},
			wantCode:   "RENVO-CHECK-028",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid operand type for builtin",
			wantLine:   2,
			wantColumn: 26,
		},
		{
			name:       "non_function_call",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { x := 1; x() }\n"},
			wantCode:   "RENVO-CHECK-011",
			wantFile:   "cmd/app/main.go",
			wantDetail: "called expression is not a function",
		},
		{
			name:       "assignment_target",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { 1 = 2 }\n"},
			wantCode:   "RENVO-CHECK-012",
			wantFile:   "cmd/app/main.go",
			wantDetail: "left side of assignment is not assignable",
		},
		{
			name:       "assignment_count",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { a, b := 1; _, _ = a, b }\n"},
			wantCode:   "RENVO-CHECK-013",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment count does not match",
		},
		{
			name:       "break_placement",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { break }\n"},
			wantCode:   "RENVO-CHECK-014",
			wantFile:   "cmd/app/main.go",
			wantDetail: "break is not inside a loop or switch",
		},
		{
			name:       "continue_placement",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { continue }\n"},
			wantCode:   "RENVO-CHECK-015",
			wantFile:   "cmd/app/main.go",
			wantDetail: "continue is not inside a loop",
		},
	}

	for _, frontend := range frontends {
		frontend := frontend
		for _, tc := range cases {
			tc := tc
			t.Run(frontend.name+"/"+tc.name, func(t *testing.T) {
				runFrontendDiagnosticCase(t, frontend.config, tc, nil)
			})
		}
	}
}

func TestFrontendBackendDiagnosticPreservesDetail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backend failure helper uses a POSIX shell script")
	}
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if frontend.backend == "" {
		t.Skip("selected standalone frontend uses its embedded backend")
	}
	dir := t.TempDir()
	backend := filepath.Join(dir, "backend-failure")
	if err := os.WriteFile(backend, []byte("#!/bin/sh\necho 'intentional backend failure' >&2\nexit 23\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tc := frontendDiagnosticCase{
		name:       "backend",
		files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() {}\n"},
		wantCode:   "RENVO-BACKEND-003",
		wantDetail: "intentional backend failure",
	}
	frontend.backend = backend
	runFrontendDiagnosticCase(t, frontend, tc, nil)
}

func TestFrontendStructuredOptionDiagnostics(t *testing.T) {
	root := repoRoot(t)
	frontends := []struct {
		name   string
		config frontendConfig
	}{
		{name: "host", config: frontendCompiler(t, root)},
		{name: "stage3", config: selfHostedFrontendCompiler(t, root)},
	}
	cases := []struct {
		name string
		args []string
		code string
	}{
		{name: "script_requires_file", args: []string{"-script", "-o", "app", "."}, code: "RENVO-OPTION-033"},
		{name: "script_file_count", args: []string{"-script", "-o", "app", "first.go", "second.go"}, code: "RENVO-OPTION-034"},
		{name: "conflicting_emit", args: []string{"-emit-unit", "-emit-image", "-o", "app", "first.go"}, code: "RENVO-OPTION-035"},
	}
	for _, frontend := range frontends {
		for _, tc := range cases {
			t.Run(frontend.name+"/"+tc.name, func(t *testing.T) {
				dir := t.TempDir()
				for _, name := range []string{"first.go", "second.go"} {
					if err := os.WriteFile(filepath.Join(dir, name), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				cmd := frontendCommand(frontend.config, tc.args...)
				cmd.Dir = dir
				cmd.Env = frontendCommandEnv(frontend.config.env, dir)
				out, err := cmd.CombinedOutput()
				if err == nil {
					t.Fatalf("frontend unexpectedly accepted options %q", tc.args)
				}
				text := string(out)
				if strings.Count(text, "error RENVO-") != 1 || !strings.Contains(text, "error "+tc.code+" ") {
					t.Fatalf("diagnostic = %q, want exactly one %s error", text, tc.code)
				}
			})
		}
	}
}

func runFrontendDiagnosticCase(t *testing.T, frontend frontendConfig, tc frontendDiagnosticCase, envOverride []string) {
	t.Helper()
	if frontend.compiler == "" {
		t.Fatal("frontend compiler is unavailable")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/diagnostic\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, source := range tc.files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	output := filepath.Join(dir, "app")
	cmd := frontendCommand(frontend, "-t", frontend.target, "-s", "-o", output, "./cmd/app")
	cmd.Dir = dir
	env := append([]string(nil), frontend.env...)
	for _, override := range envOverride {
		env = setFrontendEnv(env, override)
	}
	cmd.Env = frontendCommandEnv(env, dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("frontend unexpectedly accepted %s", tc.name)
	}
	text := string(out)
	if strings.Count(text, "error RENVO-") != 1 {
		t.Fatalf("diagnostic = %q, want exactly one structured error", text)
	}
	if strings.Contains(text, "frontend compilation failed") {
		t.Fatalf("diagnostic fell back to generic frontend failure: %q", text)
	}
	if strings.Contains(text, "renvo: compilation failed") || strings.Contains(text, "renvo: wasm32 compilation failed") {
		t.Fatalf("diagnostic included a generic backend failure: %q", text)
	}
	if !strings.Contains(text, "error "+tc.wantCode+" ") {
		t.Fatalf("diagnostic = %q, want stable code %s", text, tc.wantCode)
	}
	if !strings.Contains(text, tc.wantDetail) {
		t.Fatalf("diagnostic = %q, want detail %q", text, tc.wantDetail)
	}
	if tc.wantFile != "" {
		wantPath := filepath.Join(dir, filepath.FromSlash(tc.wantFile))
		line, column := frontendSourceLocation(t, text, wantPath)
		if tc.wantLine > 0 && (line != tc.wantLine || column != tc.wantColumn) {
			t.Fatalf("diagnostic = %q, location = %d:%d, want %d:%d", text, line, column, tc.wantLine, tc.wantColumn)
		}
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("failed compilation left output %q (stat error %v)", output, statErr)
	}
}

func frontendSourceLocation(t *testing.T, diagnostic string, wantPath string) (int, int) {
	t.Helper()
	prefix := wantPath + ":"
	start := strings.Index(diagnostic, prefix)
	if start < 0 {
		t.Fatalf("diagnostic = %q, want source location beginning %q", diagnostic, prefix)
	}
	remainder := diagnostic[start+len(prefix):]
	lineEnd := strings.IndexByte(remainder, ':')
	if lineEnd < 1 {
		t.Fatalf("diagnostic = %q, want a line after %q", diagnostic, prefix)
	}
	columnEnd := strings.IndexByte(remainder[lineEnd+1:], ':')
	if columnEnd < 1 {
		t.Fatalf("diagnostic = %q, want a column after %q", diagnostic, prefix)
	}
	line, lineErr := strconv.Atoi(remainder[:lineEnd])
	column, columnErr := strconv.Atoi(remainder[lineEnd+1 : lineEnd+1+columnEnd])
	if lineErr != nil || columnErr != nil || line < 1 || column < 1 {
		t.Fatalf("diagnostic = %q, want positive line and column for %q", diagnostic, wantPath)
	}
	return line, column
}

func setFrontendEnv(env []string, item string) []string {
	key := envKey(item)
	for i := 0; i < len(env); i++ {
		if envKey(env[i]) == key {
			env[i] = item
			return env
		}
	}
	return append(env, item)
}
