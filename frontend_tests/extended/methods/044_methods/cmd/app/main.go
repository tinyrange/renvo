package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 13
	if int(c.add(10)) == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
