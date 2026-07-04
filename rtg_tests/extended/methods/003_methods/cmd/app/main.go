package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 3
	if int(c.add(3)) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
