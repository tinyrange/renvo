package box

type Hidden interface{ boxed() int }
type Value int

func (v Value) boxed() int { return int(v) }

func (v Value) accumulate(values ...int) int {
	result := int(v)
	for _, value := range values {
		result += value
	}
	return result
}

type Accumulator interface{ accumulate(...int) int }
type VariadicHolder struct {
	Padding int
	Accumulator
}

func Variadic[T interface{ accumulate(...int) int }](v T, values ...int) int {
	return v.accumulate(values...) + v.accumulate(1, 2) + v.accumulate()
}

func VariadicExpression[T interface{ accumulate(...int) int }]() func(T, ...int) int {
	return T.accumulate
}

type other int

func (v other) hidden() int { return int(v) }

type Mixed struct {
	Padding int
	Hidden
	other
}

func New(a, b int) Mixed { return Mixed{11, Value(a), other(b)} }

func Call[T interface {
	boxed() int
	hidden() int
}](v T) int {
	return v.boxed() + v.hidden()
}

func Bound[T interface{ boxed() int }](v T) func() int { return v.boxed }

type Nested[T any] struct {
	Padding T
	*Mixed
}

func (n Nested[T]) Sum() int { return Call(n) }

func Assert(v any) int {
	if view, ok := v.(interface {
		boxed() int
		hidden() int
	}); ok {
		return Call(view)
	}
	return -1
}

func Expression[T interface{ boxed() int }]() func(T) int { return T.boxed }

func ConcreteExpression() func(Mixed) int { return Mixed.boxed }

func CheckNilMethodValues() bool {
	nilInterfaceFailed := false
	func() {
		defer func() { nilInterfaceFailed = recover() != nil }()
		var receiver Hidden
		_ = Bound(receiver)
	}()
	if !nilInterfaceFailed {
		return false
	}
	var pointer *Value
	bound := Bound(pointer)
	if bound == nil {
		return false
	}
	nilPointerFailed := false
	func() {
		defer func() { nilPointerFailed = recover() != nil }()
		bound()
	}()
	return nilPointerFailed
}
