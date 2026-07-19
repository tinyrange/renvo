package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 10
	if int(c.add(7)) == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
