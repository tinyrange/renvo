package callback

type Apply func(func(value int) int) int
type Value func(value int) int
type Values []Value

func Run(fn func(value int) int) int                    { return fn(42) }
func Return(fn func(value int) int) func(other int) int { return fn }
func Slice(values []func(value int) int) int            { return values[0](42) }
