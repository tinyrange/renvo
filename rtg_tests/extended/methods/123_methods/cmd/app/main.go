package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 30
	if int(c.add(4)) == 34 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
