package main

type dynamicDispatchPair struct {
	left  int
	right int
}

type dynamicDispatchScorer interface {
	Score(int) int
	Pair() dynamicDispatchPair
	Split() (int, bool)
	Touch(int)
}

type dynamicDispatchNarrow interface {
	Score(int) int
}

type dynamicDispatchValue struct {
	base int
}

var dynamicDispatchTouched int

func (value dynamicDispatchValue) Score(extra int) int {
	return value.base + extra
}

func (value dynamicDispatchValue) Pair() dynamicDispatchPair {
	return dynamicDispatchPair{left: value.base, right: value.base + 1}
}

func (value dynamicDispatchValue) Split() (int, bool) {
	result := value.base + 2
	return result, true
}

func (value dynamicDispatchValue) Touch(extra int) {
	dynamicDispatchTouched = value.base + extra
}

type dynamicDispatchPointer struct {
	base int
}

func (value *dynamicDispatchPointer) Score(extra int) int {
	return value.base*2 + extra
}

func (value *dynamicDispatchPointer) Pair() dynamicDispatchPair {
	return dynamicDispatchPair{left: value.base * 2, right: value.base * 3}
}

func (value *dynamicDispatchPointer) Split() (int, bool) {
	result := value.base * 4
	return result, true
}

func (value *dynamicDispatchPointer) Touch(extra int) {
	dynamicDispatchTouched = value.base*2 + extra
}

type dynamicDispatchUnrelated struct{}

func (dynamicDispatchUnrelated) Score(extra int) string {
	return "unrelated"
}

func dynamicDispatchChoose(first bool) dynamicDispatchScorer {
	if first {
		return dynamicDispatchValue{base: 10}
	}
	return &dynamicDispatchPointer{base: 20}
}

func dynamicDispatchVerify(value dynamicDispatchScorer, score int, left int, right int, split int) bool {
	if value.Score(3) != score {
		print("FAIL score\n")
		return false
	}
	pair := value.Pair()
	if pair.left != left || pair.right != right {
		print("FAIL pair\n")
		return false
	}
	part, ok := value.Split()
	if !ok || part != split {
		print("FAIL split\n")
	}
	return ok && part == split
}

func dynamicDispatchDefer(value dynamicDispatchScorer) {
	defer value.Touch(7)
}

func appMain() int {
	first := dynamicDispatchChoose(true)
	second := dynamicDispatchChoose(false)
	value := dynamicDispatchValue{base: 30}
	var pointerToValue dynamicDispatchScorer = &value
	if !dynamicDispatchVerify(first, 13, 10, 11, 12) {
		print("FAIL first\n")
		return 1
	}
	if !dynamicDispatchVerify(second, 43, 40, 60, 80) {
		print("FAIL second\n")
		return 1
	}
	if !dynamicDispatchVerify(pointerToValue, 33, 30, 31, 32) {
		print("FAIL pointer\n")
		return 1
	}
	values := []dynamicDispatchScorer{first, second, pointerToValue}
	if values[0].Score(1) != 11 || values[1].Score(1) != 41 || values[2].Score(1) != 31 {
		print("FAIL slice\n")
		return 1
	}
	var narrow dynamicDispatchNarrow = second
	if narrow.Score(4) != 44 {
		print("FAIL narrow\n")
		return 1
	}
	dynamicDispatchDefer(pointerToValue)
	if dynamicDispatchTouched != 37 {
		print("FAIL defer\n")
		return 1
	}
	print("PASS\n")
	return 0
}
