package main

func increment(value int) int                      { return value + 1 }
func invoke(callback func(int) int, value int) int { return callback(value) }

func main() {
	var callback func(int) int
	callback = increment
	if invoke(callback, 2) != 3 {
		panic("named function")
	}
	total := 10
	callback = func(value int) int { total += value; return total }
	stored := []func(int) int{callback}
	total = 20
	if invoke(callback, 2) != 22 || stored[0](3) != 25 || total != 25 {
		panic("shared closure capture")
	}
	callback = increment
	if callback(4) != 5 || stored[0](1) != 26 {
		panic("closure copy")
	}
	println("PASS")
}
