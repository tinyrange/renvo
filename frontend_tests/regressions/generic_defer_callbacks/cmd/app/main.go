package main

type Callback[V any] func()
type Pair[V any] func(V, string)
type Variadic[V any] func(...V)
type TupleVariadic[V any] func(V, ...V)
type FixedTupleVariadic[V any] func(V, V, ...V)
type AnyTupleVariadic[V any] func(V, ...any)

func caught() (ok bool) {
	var callback Callback[int] = func() { ok = recover() != nil }
	defer callback()
	panic("expected")
}

func indirect() any { return recover() }
func indirectCaught() (ok bool) {
	defer func() { ok = recover() != nil }()
	var callback Callback[int] = func() {
		if indirect() != nil {
			panic("indirect recover succeeded")
		}
	}
	defer callback()
	panic("expected")
}

func nestedCaught() (outer, inner bool) {
	defer func() { outer = recover() != nil }()
	var callback Callback[int] = func() {
		defer func() { inner = recover() != nil }()
	}
	defer callback()
	panic("original")
}

type Catcher struct{ Result *bool }

func (c Catcher) Catch() { *c.Result = recover() != nil }

type Recovery interface{ Catch() }

func methodCaught(dynamic bool) (ok bool) {
	c := Catcher{Result: &ok}
	var callback Callback[int]
	if dynamic {
		var receiver Recovery = c
		callback = receiver.Catch
	} else {
		callback = c.Catch
	}
	defer callback()
	panic("method")
}

func snapshots() {
	events := ""
	factory := func() Pair[int] {
		events += "1"
		return func(n int, text string) {
			if n != 42 || text != "saved" {
				panic("argument snapshots")
			}
			events += "4"
		}
	}
	argument := func() int { events += "2"; return 42 }
	work := func() {
		defer factory()(argument(), "saved")
		events += "3"
	}
	work()
	if events != "1234" {
		panic("callee and argument evaluation order")
	}
	seen := ""
	pair := func() (int, string) { seen += "a"; return 7, "b" }
	var callback Pair[int] = func(n int, text string) {
		if n != 7 || text != "b" {
			panic("tuple arguments")
		}
		seen += "c"
	}
	work = func() { defer callback(pair()); seen += "b" }
	work()
	if seen != "abc" {
		panic("tuple evaluation")
	}
}

func variadic() {
	count := 0
	var callback Variadic[int] = func(items ...int) {
		count++
		if len(items) == 0 {
			if items != nil {
				panic("empty variadic arguments are not nil")
			}
		} else if len(items) != 2 || items[0] != 5 || items[1] != 6 {
			panic("variadic values")
		}
	}
	callback()
	callback(5, 6)
	callback([]int{5, 6}...)
	work := func() {
		defer callback()
		defer callback(5, 6)
		items := []int{4, 6}
		defer callback(items...)
		items[0] = 5
	}
	work()
	if count != 6 {
		panic("variadic defer count")
	}
}

func nilCaught() (ok bool) {
	registered := false
	defer func() { ok = recover() != nil && registered }()
	var callback Callback[int]
	defer callback()
	registered = true
	return
}

func tupleVariadics() {
	events := ""
	pair := func() (int, int) { events += "p"; return 42, 43 }
	var callback TupleVariadic[int] = func(first int, rest ...int) {
		if first != 42 || len(rest) != 1 || rest[0] != 43 {
			panic("tuple variadic values")
		}
		events += "d"
	}
	factory := func() TupleVariadic[int] { events += "f"; return callback }
	work := func() { defer factory()(pair()); events += "b" }
	work()
	if events != "fpbd" {
		panic("tuple callee and argument evaluation")
	}
	count := 0
	var all Variadic[int] = func(items ...int) {
		if len(items) != 2 || items[0] != 42 || items[1] != 43 {
			panic("all variadic tuple")
		}
		count++
	}
	var fixed FixedTupleVariadic[int] = func(first, second int, rest ...int) {
		if first != 42 || second != 43 || rest != nil {
			panic("tuple without variadic values")
		}
		count++
	}
	mixed := func() (int, string, int) { return 42, "value", 43 }
	var boxed AnyTupleVariadic[int] = func(first int, rest ...any) {
		if first != 42 || len(rest) != 2 || rest[0].(string) != "value" || rest[1].(int) != 43 {
			panic("boxed tuple variadic values")
		}
		count++
	}
	work = func() { defer all(pair()); defer fixed(pair()); defer boxed(mixed()) }
	work()
	if count != 3 {
		panic("tuple variadic defer count")
	}
}

func main() {
	if !caught() || !indirectCaught() || !methodCaught(false) || !methodCaught(true) || !nilCaught() {
		panic("deferred callback recovery")
	}
	outer, inner := nestedCaught()
	if !outer || inner {
		panic("normal nested return recovered an outer panic")
	}
	snapshots()
	variadic()
	tupleVariadics()
	print("PASS\n")
}
