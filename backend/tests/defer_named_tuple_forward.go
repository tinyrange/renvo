package main

type result struct {
	n     int
	extra int
}
type problem struct{ message string }

func (p *problem) Error() string { return p.message }

var calls int

func producer() (result, *problem) {
	calls++
	return result{n: 40, extra: 7}, &problem{message: "expected"}
}

func adjust(value *result) { value.n += 2 }

func forward() (value result, err error) {
	defer adjust(&value)
	return producer()
}

func appMain(args []string) int {
	value, err := forward()
	if calls != 1 || value.n != 42 || value.extra != 7 || err == nil || err.Error() != "expected" {
		return 1
	}
	print("PASS\n")
	return 0
}
