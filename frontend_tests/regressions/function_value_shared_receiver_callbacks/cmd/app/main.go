package main

type handler func()

type button struct {
	click handler
}

type controller struct {
	total int
}

func (c *controller) one() {
	c.total++
}

func (c *controller) two() {
	c.total += 2
}

func main() {
	c := &controller{}
	var first button
	var second button
	first.click = c.one
	second.click = c.two
	first.click()
	second.click()
	if c.total == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
