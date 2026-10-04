package main

type Seq[V any] func(func(V) bool)
type Callback[V any] func(V, ...any)

func values[V any](items ...V) Seq[V] {
	return func(yield func(V) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

func add(events *string, value string) { *events += value }
func pair() (int, string, int)         { return 42, "x", 43 }

func ordered[V any](sequence Seq[V], events *string) (result int) {
	defer add(events, "o")
	for value := range sequence {
		defer func(captured V) { _ = captured; result++; *events += "v" }(value)
		defer add(events, "d")
		if result != 0 {
			panic("early defer")
		}
	}
	defer add(events, "a")
	*events += "b"
	return 10
}

func returned[V any](sequence Seq[V], events *string) (result int, other string) {
	defer func() { result += 2; other += "d" }()
	for range sequence {
		defer func() { result++; *events += "d" }()
		result := 42
		return result, "x"
	}
	return 0, ""
}

func snapshots[V any](sequence Seq[V], events *string) {
	factory := func() Callback[int] {
		*events += "f"
		return func(value int, rest ...any) {
			if value != 42 || len(rest) != 2 || rest[0].(string) != "x" || rest[1].(int) != 43 {
				panic("tuple snapshot")
			}
			*events += "t"
		}
	}
	for value := range sequence {
		defer factory()(pair())
		defer func() { _ = value; *events += "c" }()
	}
	*events += "b"
}

func recovered(events *string) {
	value := recover()
	if value == nil {
		*events += "n"
	} else {
		*events += value.(string)
	}
}
func helper(events *string) { defer recovered(events) }

//go:noinline
func invoke(callback func()) { callback() }

func panicking[V any](sequence Seq[V], events *string) {
	defer add(events, "o")
	for range sequence {
		defer recovered(events)
		defer helper(events)
		panic("p")
	}
}

func replacing[V any](sequence Seq[V], events *string) {
	defer recovered(events)
	for range sequence {
		defer recovered(events)
		defer func() { panic("q") }()
		panic("p")
	}
}

func nested[V any](sequence Seq[V], events *string) {
	for range sequence {
		defer add(events, "o")
		for range sequence {
			defer add(events, "i")
		}
		invoke(func() {
			for range sequence {
				defer add(events, "l")
			}
			*events += "b"
		})
		*events += "a"
	}
}

func builtins[V any](sequence Seq[V], data map[int]int, dst []int) {
	for range sequence {
		defer delete(data, 42)
		defer copy(dst, []int{43})
		defer recover()
	}
	if data[42] != 42 || dst[0] != 0 {
		panic("early builtin")
	}
}

type Method[V any] struct{ events *string }

func (m *Method[V]) Recover() {
	if value := recover(); value != nil {
		*m.events += value.(string)
	}
}

type Recoverer interface{ Recover() }
type RecovererAlias = Recoverer

func nilInterface[V any](sequence Seq[V], method RecovererAlias, events *string) {
	defer func() {
		if recover() != nil {
			*events += "r"
		}
	}()
	for range sequence {
		defer method.Recover()
		*events += "b"
	}
}

func nilMethodValue[V any](value V, method Recoverer, events *string) {
	defer func() {
		if recover() != nil {
			*events += "r"
		}
	}()
	var callback func() = method.Recover
	_ = value
	*events += "b"
	callback()
}

type NilMethod[V any] struct{}

func (m *NilMethod[V]) Recover() {
	if m != nil {
		panic("typed nil receiver")
	}
}

func discardedRecover[V any](value V) {
	recover()
	_ = value
}

func discardedParenthesizedRecover[V any](value V) {
	(recover())
	_ = value
}

func discardedInitializerRecover[V any](value V) {
	if recover(); true {
		_ = value
	}
}

func discardedPostRecover[V any](value V) {
	for count := 0; count < 1; recover() {
		count++
	}
	_ = value
}

func discardedRecoverCases(events *string) {
	func() { defer discardedRecover(42); panic("p") }()
	func() { defer discardedParenthesizedRecover(42); panic("p") }()
	func() { defer discardedInitializerRecover(42); panic("p") }()
	func() { defer discardedPostRecover(42); panic("p") }()
	func() {
		defer func() {
			if recover() != nil {
				*events += "r"
			}
		}()
		recover := func() (int, int) { *events += "s"; return 42, 43 }
		recover()
		panic("p")
	}()
}

func methods[V any](sequence Seq[V], method Recoverer, events *string) {
	for range sequence {
		defer method.Recover()
		defer add(events, "d")
	}
	panic("p")
}

func nilCallback[V any](sequence Seq[V], events *string) {
	defer func() {
		if recover() != nil {
			*events += "r"
		}
	}()
	for range sequence {
		var callback func()
		defer callback()
		*events += "b"
	}
}

func returnPanic[V any](sequence Seq[V], events *string) {
	defer recovered(events)
	for range sequence {
		defer recovered(events)
		defer func() { panic("q") }()
	}
}

func innerPanic(events *string) {
	defer recovered(events)
	panic("h")
}

func resumed[V any](sequence Seq[V], events *string) {
	for range sequence {
		defer recovered(events)
		defer innerPanic(events)
		panic("p")
	}
}

func iteratorPanic[V any](value V, events *string) {
	for range func(yield func(V) bool) { yield(value); panic("i") } {
		defer recovered(events)
		defer add(events, "d")
	}
}

func main() {
	events := ""
	if result := ordered(values(42, 43), &events); result != 12 || events != "badvdvo" {
		panic("order:" + events)
	}
	events = ""
	result, other := returned(values(42), &events)
	if result != 45 || other != "xd" || events != "d" {
		panic("return")
	}
	events = ""
	snapshots(values(42, 43), &events)
	if events != "ffbctct" {
		panic("snapshots:" + events)
	}
	events = ""
	panicking(values(42), &events)
	if events != "npo" {
		panic("recover:" + events)
	}
	events = ""
	replacing(values(42), &events)
	if events != "qn" {
		panic("replacement:" + events)
	}
	events = ""
	nested(values(42), &events)
	if events != "blaio" {
		panic("nested:" + events)
	}
	data := map[int]int{42: 42}
	dst := []int{0}
	builtins(values(42), data, dst)
	if len(data) != 0 || dst[0] != 43 {
		panic("builtins")
	}
	events = ""
	methods(values(42), &Method[int]{events: &events}, &events)
	if events != "dp" {
		panic("method:" + events)
	}
	events = ""
	nilCallback(values(42), &events)
	if events != "br" {
		panic("nil:" + events)
	}
	events = ""
	returnPanic(values(42), &events)
	if events != "qn" {
		panic("return panic:" + events)
	}
	events = ""
	resumed(values(42), &events)
	if events != "hp" {
		panic("resumed panic:" + events)
	}
	events = ""
	iteratorPanic(42, &events)
	if events != "di" {
		panic("iterator panic:" + events)
	}
	events = ""
	nilInterface(values(42), nil, &events)
	if events != "r" {
		panic("nil interface timing:" + events)
	}
	events = ""
	nilMethodValue(42, nil, &events)
	if events != "r" {
		panic("nil method value timing:" + events)
	}
	events = ""
	var nilReceiver *NilMethod[int]
	nilInterface(values(42), nilReceiver, &events)
	if events != "b" {
		panic("typed nil interface:" + events)
	}
	events = ""
	discardedRecoverCases(&events)
	if events != "sr" {
		panic("discarded recover:" + events)
	}
	print("PASS\n")
}
