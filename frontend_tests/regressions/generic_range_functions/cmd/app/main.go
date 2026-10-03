package main

import "example.com/rangefunctions/sequence"

type Seq[V any] func(func(V) bool)
type Yield[V any] func(V) bool
type NamedSeq[V any] func(Yield[V])
type ParenthesizedSeq[V any] func(Yield[V])

type Iterator[V any] interface{ Each(func(V) bool) }
type IteratorAlias = Iterator[int]
type EmbeddedIterator interface{ IteratorAlias }

type Source[V any] struct{ Value V }

func (source Source[V]) Each(yield func(V) bool) { yield(source.Value) }

func values[V any](items ...V) Seq[V] {
	return func(yield func(V) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

func named[V any](item V) NamedSeq[V] {
	return func(yield Yield[V]) { yield(item) }
}

func direct(yield func(int) bool) {
	if yield(2) {
		yield(4)
	}
}

func bindings() {
	count := 0
	factory := func() Seq[int] { count++; return values(1, 2, 3) }
	var callbacks []func() int
	var pointers []*int
	for v := range factory() {
		callbacks = append(callbacks, func() int { return v })
		pointers = append(pointers, &v)
	}
	if count != 1 || len(callbacks) != 3 {
		panic("iterator expression evaluation")
	}
	for i, callback := range callbacks {
		if callback() != i+1 || *pointers[i] != i+1 {
			panic("fresh range bindings")
		}
	}
	x := 0
	for x = range values(5, 6) {
		x++
	}
	if x != 7 {
		panic("assigned range binding")
	}
	for x := range named(42) {
		if x != 42 {
			panic("named yield")
		}
	}
	total := 0
	for n := range direct {
		total += n
	}
	for n := range func(yield func(int) bool) { yield(3) } {
		total += n
	}
	for n := range func() Seq[int] { return values(5) }() {
		total += n
	}
	if total != 14 {
		panic("direct iterator values")
	}
	visits := 0
	for range func(yield func() bool) { yield(); yield() } {
		visits++
	}
	if visits != 2 {
		panic("zero yields")
	}
	for _ = range values(1, 2) {
		visits++
	}
	if visits != 4 {
		panic("blank assigned binding")
	}
	for item := range sequence.Items(42) {
		if item.N != 42 {
			panic("private cross-package yield type")
		}
		var boxed any = item
		if _, ok := boxed.(struct{ N int }); ok {
			panic("private yield type identity")
		}
	}
	source := Source[int]{Value: 42}
	for item := range source.Each {
		if item != 42 {
			panic("generic iterator method value")
		}
	}
	var iterator EmbeddedIterator = source
	for item := range iterator.Each {
		if item != 42 {
			panic("embedded interface iterator method")
		}
	}
	var anonymous interface{ Each(func(int) bool) } = source
	for item := range anonymous.Each {
		if item != 42 {
			panic("anonymous interface iterator method")
		}
	}
	var parenthesized ParenthesizedSeq[int] = func(yield Yield[int]) { yield(42) }
	for item := range parenthesized {
		if item != 42 {
			panic("parenthesized named yield type")
		}
	}
	index := 0
	next := func() int { index++; return index - 1 }
	array := make([]int, 2)
	n := 0
	for array[next()], n = range func(yield func(int, int) bool) { yield(3, 4); yield(5, 6) } {
		if n != 4 && n != 6 {
			panic("two-value assignment")
		}
	}
	if index != 2 || array[0] != 3 || array[1] != 5 || n != 6 {
		panic("assignment destination evaluation")
	}
}

func branches() {
	total := 0
	for n := range values(1, 2, 3, 4) {
		switch n {
		case 1:
			continue
		case 2:
			break
		}
		for i := 0; i < 3; i++ {
			if i == 1 {
				continue
			}
			if i == 2 {
				break
			}
			total++
		}
		for _, unused := range []int{1, 2} {
			_ = unused
			continue
		}
		if n == 3 {
			break
		}
	}
	if total != 2 {
		panic("nested native branches")
	}
	total = 0
Outer:
	for i := 0; i < 3; i++ {
		for n := range values(1, 2, 3) {
			if n == 2 {
				continue Outer
			}
			total++
		}
	}
	if total != 3 {
		panic("continue outer native loop")
	}
	total = 0
Range:
	for n := range values(1, 2, 3) {
		for m := range values(1, 2) {
			if m == 2 {
				continue Range
			}
			total += n
		}
	}
	if total != 6 {
		panic("continue outer function range")
	}
Stop:
	for range values(1, 2, 3) {
		for range values(1, 2) {
			break Stop
		}
		panic("missed labelled break")
	}
	for n := range values(1, 2) {
		keyed := struct{ Done int }{Done: n}
		_ = keyed
		local := func() {
			goto Done
		Done:
			return
		}
		local()
		if n == 1 {
			goto Skip
		}
		panic("missed local goto")
	Skip:
		goto Done
	}
	panic("missed outward goto")
Done:
}

func result(events *string) (n int) {
	seq := func(yield func(int) bool) {
		defer func() { *events += "c"; *events += string(rune('0' + n)) }()
		yield(7)
		*events += "i"
	}
	for n := range seq {
		return n
	}
	return -1
}

func tuple() (int, string) {
	for n := range values(1, 2, 3) {
		for s := range values("a", "b") {
			if n == 2 && s == "b" {
				return n + 40, s
			}
		}
	}
	return 0, ""
}

func bare() (n, m int) {
	n, m = 4, 5
	for range values(0) {
		return
	}
	return -1, -1
}

func caught(f func()) bool {
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		f()
	}()
	return panicked
}

func protocol() {
	var again func() bool
	if !caught(func() {
		for range func(yield func() bool) { again = yield; yield() } {
			again()
		}
	}) {
		panic("reentrant yield")
	}
	var absent Seq[int]
	if !caught(func() {
		for range absent {
		}
	}) {
		panic("nil iterator")
	}
	if !caught(func() {
		for range func(yield func() bool) { yield(); yield() } {
			break
		}
	}) {
		panic("yield after false")
	}
	var saved func() bool
	for range func(yield func() bool) { saved = yield } {
		panic("unexpected body")
	}
	if !caught(func() { saved() }) {
		panic("yield after exhaustion")
	}
	if !caught(func() {
		for range func(yield func() bool) {
			func() { defer func() { _ = recover() }(); yield() }()
		} {
			panic("body")
		}
	}) {
		panic("swallowed body panic")
	}
}

func main() {
	bindings()
	branches()
	events := ""
	if result(&events) != 7 || events != "ic7" {
		panic("named result assignment before iterator cleanup")
	}
	n, s := tuple()
	if n != 42 || s != "b" {
		panic("nested tuple return")
	}
	n, m := bare()
	if n != 4 || m != 5 {
		panic("bare return")
	}
	protocol()
	print("PASS\n")
}
