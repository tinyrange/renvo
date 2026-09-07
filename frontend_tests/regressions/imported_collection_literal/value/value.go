package value

type Value struct{ n int }

func New(n int) Value       { return Value{n} }
func (v Value) Number() int { return v.n }
