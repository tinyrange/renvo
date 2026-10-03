package callback

type Base struct{ Value int }
type Arg = struct{ Base }
type PointerArg = struct{ *Base }
type GenericBase[T any] struct{ Value T }
type GenericArg[T any] = struct{ GenericBase[T] }
type base struct{ Value int }
type PrivateBase = base
type PrivateArg = struct{ base }

func Apply(f func(Arg) int) int               { return f(Arg{Base{42}}) }
func ApplyPointer(f func(PointerArg) int) int { return f(PointerArg{&Base{43}}) }
func ApplyGeneric(f func(GenericArg[int]) int) int {
	return f(GenericArg[int]{GenericBase[int]{44}})
}
func Box() any        { return Arg{Base{45}} }
func BoxPointer() any { return PointerArg{&Base{46}} }
func BoxGeneric() any { return GenericArg[int]{GenericBase[int]{47}} }
func BoxPrivate() any { return PrivateArg{base{48}} }
