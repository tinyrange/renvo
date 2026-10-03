package callback

type Scalar = int
type Arg = struct{ Value Scalar }
type First interface{ First() int }
type AnotherFirst interface{ First() int }
type Reader = interface {
	First
	AnotherFirst
	Second() int
}
type NamedScalar int
type PrivateInner = struct{ value int }
type PrivateArg = struct{ Value PrivateInner }

func Apply(f func(Arg) int) int                    { return f(Arg{42}) }
func ApplyReader(f func(Reader) int, v Reader) int { return f(v) }
func Sum(v Arg) int                                { return v.Value }
func ApplyPrivate(f func(PrivateArg) int) int      { return f(PrivateArg{PrivateInner{42}}) }
func PrivateValue(v PrivateInner) int              { return v.value }

const ArrayLength = 2

type ArrayArg = struct{ Value [ArrayLength]int }

func ApplyArray(f func(ArrayArg) int) int { return f(ArrayArg{[ArrayLength]int{20, 22}}) }
