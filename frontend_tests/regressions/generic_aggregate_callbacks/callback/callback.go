package callback

type Arg = struct{ Value int }
type Private = struct{ value int }
type Reader = interface{ Value() int }

func Apply(f func(Arg) Arg) int                    { return f(Arg{42}).Value }
func ApplyPrivate(f func(Private) Private) int     { return f(Private{42}).value }
func ApplyReader(f func(Reader) int, v Reader) int { return f(v) }
func PrivateValue(v Private) int                   { return v.value }
