package callback

type Arg = struct{ A, B int }
type Tagged = struct {
	Value int "json:\"value\""
}
type Reader = interface {
	First() int
	Second() int
}

func Apply(f func(Arg) int) int                    { return f(Arg{20, 22}) }
func ApplyReader(f func(Reader) int, v Reader) int { return f(v) }
func ApplyTagged(f func(Tagged) int) int           { return f(Tagged{42}) }
func Sum(v Arg) int                                { return v.A + v.B }
