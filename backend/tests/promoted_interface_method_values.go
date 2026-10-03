package main

type promotedResult struct {
	text    string
	numbers [2]int
}

type promotedMethodReceiver struct{ value int }

func (v *promotedMethodReceiver) update(text string, numbers ...int) promotedResult {
	for i := 0; i < len(numbers); i++ {
		v.value += numbers[i]
	}
	return promotedResult{text, [2]int{v.value, len(numbers)}}
}

type promotedMethodContainer struct {
	padding int
	promotedMethodReceiver
}

func appMain(args []string) int {
	v := promotedMethodContainer{padding: 11, promotedMethodReceiver: promotedMethodReceiver{value: 3}}
	var receiver interface {
		update(string, ...int) promotedResult
	} = &v
	bound := receiver.update
	result := bound("first", 4, 5)
	if result.text != "first" || result.numbers[0] != 12 || result.numbers[1] != 2 || v.value != 12 {
		return 1
	}
	v.promotedMethodReceiver = promotedMethodReceiver{value: 20}
	result = bound("second", []int{6, 7}...)
	if result.text != "second" || result.numbers[0] != 33 || result.numbers[1] != 2 || v.value != 33 {
		return 2
	}
	print("PASS\n")
	return 0
}
