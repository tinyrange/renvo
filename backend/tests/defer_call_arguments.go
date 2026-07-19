package main

type deferredPair struct {
	a int
	b int
}

type deferredRecorder struct {
	base int
	more int
}

type deferredPutter interface {
	Put(string, []int, deferredPair)
}

var deferredTrace int
var deferredTotal int

func deferredNext(v int) int {
	deferredTrace = deferredTrace*10 + v
	return v
}

func deferredChoose() func(int) {
	deferredTrace = deferredTrace*10 + 1
	return deferredRecord
}

func deferredRecord(v int) {
	deferredTrace = deferredTrace*10 + v
}

func deferredSum(base int, values ...int) {
	deferredTotal += base
	for _, value := range values {
		deferredTotal += value
	}
}

func deferredTake(s string, values []int, value deferredPair) {
	deferredTotal += len(s) + len(values) + values[0] + value.a + value.b
}

func deferredSetResult(result *int, value int) {
	*result = value
}

func deferredNamedResult() (result int) {
	defer deferredSetResult(&result, 9)
	return 1
}

func (r deferredRecorder) Put(s string, values []int, value deferredPair) {
	deferredTotal += r.base + r.more + len(s) + len(values) + values[0] + value.a + value.b
}

func (r deferredRecorder) Add(v int) {
	deferredTotal += r.base + r.more + v
}

func (r *deferredRecorder) AddPointer(v int) {
	deferredTotal += r.base + r.more + v
}

func deferredPanicRun() {
	defer deferredRecord(8)
	panic("expected")
}

func deferredRecoverRun() (ok bool) {
	defer func() {
		if recover() != nil {
			ok = deferredTrace%10 == 8
		}
	}()
	deferredPanicRun()
	return false
}

func deferredRun() {
	values := []int{4, 5}
	value := deferredPair{a: 6, b: 7}
	defer deferredTake("xy", values, value)
	values = []int{20}
	value.a = 20

	defer deferredSum(1, 2, 3)
	defer deferredSum(10, 1)
	defer deferredSum(20, 2)
	expanded := []int{3, 4}
	defer deferredSum(30, expanded...)

	r := deferredRecorder{base: 10, more: 20}
	method := r.Add
	defer method(4)
	defer r.Add(5)
	r.base = 99
	pointerReceiver := deferredRecorder{base: 1, more: 2}
	defer pointerReceiver.AddPointer(3)
	pointerReceiver.base = 7

	var dynamic deferredPutter = deferredRecorder{base: 1, more: 2}
	dynamicValues := []int{3}
	dynamicPair := deferredPair{a: 4, b: 5}
	defer dynamic.Put("x", dynamicValues, dynamicPair)
	dynamicValues = []int{30}
	dynamicPair.a = 40

	defer deferredChoose()(deferredNext(2))
}

func appMain() int {
	deferredRun()
	if deferredTrace != 122 || deferredTotal != 195 || deferredNamedResult() != 9 || !deferredRecoverRun() {
		return 1
	}
	print("PASS\n")
	return 0
}
