package main

var global func(int) int = increment

func increment(value int) int                      { return value + 1 }
func invoke(callback func(int) int, value int) int { return callback(value) }
func returned() func(int) int                      { return global }

type counter struct{ total int }

func (c *counter) add(value int) int { c.total += value; return c.total }

func main() {
	var callback func(int) int
	if callback != nil {
		panic("zero function")
	}
	callback = increment
	if callback(2) != 3 || invoke(callback, 3) != 4 || global(4) != 5 {
		panic("named function")
	}
	stored := []func(int) int{callback, returned()}
	if stored[0](5) != 6 || stored[1](6) != 7 {
		panic("stored function")
	}
	state := counter{total: 10}
	callback = state.add
	stored[0] = callback
	if invoke(callback, 2) != 12 || stored[0](3) != 15 || state.total != 15 {
		panic("bound method")
	}
	callback = nil
	if callback != nil || stored[0](1) != 16 {
		panic("copied function")
	}
	println("PASS")
}
