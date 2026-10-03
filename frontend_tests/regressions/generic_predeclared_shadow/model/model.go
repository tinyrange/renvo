package model

var (
	int        = func(v string) string { return v + "!" }
	len        = func(v string) string { return v + "?" }
	comparable = func(v string) string { return v + "#" }
	any        = func(v string) string { return v + "@" }
	error      = func(v string) string { return v + "$" }
	new        = func() string { return "N" }
	recover    = func() string { return "R" }
	true       = 7
	false      = 9
	complex    = func() string { return "C" }
)

func Transform[T ~string](v T) T {
	return T(int(string(v)) + len(string(v)) + comparable(string(v)) + any(string(v)) + error(string(v)))
}

func Factory[T ~string]() T { return T(new() + recover()) }

func Identity[any interface{}](v any) any { return v }

func FoldedFalse[T interface{}]() bool         { return 1 == 2 }
func FoldedComplex[T interface{}]() complex128 { return 1 + 2i }
