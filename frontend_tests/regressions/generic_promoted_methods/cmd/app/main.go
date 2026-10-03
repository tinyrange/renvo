package main

import "example.com/genericpromoted/box"

func main() {
	v := box.New(42, 7)
	if box.Call(v) != 49 || box.Assert(v) != 49 {
		panic("value promotion")
	}
	nested := box.Nested[string]{Padding: "padding", Mixed: &v}
	if nested.Sum() != 49 || box.Call(&nested) != 49 || box.Assert(&nested) != 49 {
		panic("nested promotion")
	}
	expression := box.Expression[*box.Nested[string]]()
	valueExpression := box.Expression[box.Mixed]()
	concreteExpression := box.ConcreteExpression()
	if expression(&nested) != 42 || valueExpression(v) != 42 || concreteExpression(v) != 42 {
		panic("method expressions")
	}
	bound := box.Bound(&nested)
	var hidden box.Hidden = &nested
	interfaceBound := box.Bound(hidden)
	v.Hidden = box.Value(51)
	if bound() != 51 || interfaceBound() != 51 || expression(&nested) != 51 {
		panic("method value receiver")
	}
	v.Hidden = box.New(67, 19)
	if box.Call(&nested) != 74 || interfaceBound() != 67 {
		panic("embedded interface forwarding")
	}
	if !box.CheckNilMethodValues() {
		panic("generic nil method binding")
	}
	variadic := box.VariadicHolder{Padding: 99, Accumulator: box.Value(10)}
	if box.Variadic(&variadic, 3, 4) != 40 || box.Variadic(variadic) != 33 {
		panic("generic variadic call")
	}
	variadicExpression := box.VariadicExpression[*box.VariadicHolder]()
	if variadicExpression(&variadic, 5, 6) != 21 || variadicExpression(&variadic) != 10 {
		panic("variadic method expression")
	}
	print("PASS\n")
}
