package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestConstantArrayIndexValidation(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		invalid bool
	}{
		{
			name:    "high direct index",
			source:  "func check() { var values [3]int; _ = values[3] }",
			invalid: true,
		},
		{
			name:    "negative direct index",
			source:  "func check() { var values [3]int; _ = values[-1] }",
			invalid: true,
		},
		{
			name:    "folded named converted index",
			source:  "type offset int\nconst base = 1\nfunc check() { values := [4]int{}; _ = values[offset(base+3)] }",
			invalid: true,
		},
		{
			name:    "pointer parameter",
			source:  "func check(values *[2]int) { _ = values[2] }",
			invalid: true,
		},
		{
			name:    "inferred pointer local",
			source:  "func check() { values := [3]int{}; pointer := &values; _ = pointer[3] }",
			invalid: true,
		},
		{
			name:    "named array and local constant",
			source:  "const length = 3\ntype values [length]int\nfunc check() { const limit = 1+2; var data values; _ = data[limit] }",
			invalid: true,
		},
		{
			name:   "last element and dynamic index",
			source: "func check(values *[3]int, index int) { _ = values[2]; _ = values[index] }",
		},
		{
			name:   "slice constant remains runtime checked",
			source: "func check(values []int) { _ = values[100] }",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := load.SourceFile{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + tc.source + "\n")}
			program := CheckGraphCore(checkTestGraph(t, []load.SourceFile{file}))
			if tc.invalid {
				if program.Ok || program.Error != CheckErrArrayIndex {
					t.Fatalf("check result = %#v, want constant array index error", program)
				}
				if program.ErrorToken < 0 {
					t.Fatalf("diagnostic has no source token: %#v", program)
				}
				return
			}
			if !program.Ok {
				t.Fatalf("valid index rejected: %#v", program)
			}
		})
	}
}
