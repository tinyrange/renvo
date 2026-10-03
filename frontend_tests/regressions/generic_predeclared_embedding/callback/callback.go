package callback

type Arg = struct{ int }

func Id[T any](v T) T { return v }

func Box() any                  { return Arg{42} }
func Read(v Arg) int            { return v.int }
func Apply(f func(Arg) int) int { return f(Arg{43}) }
