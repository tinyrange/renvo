package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 16
	if int(c.add(7)) == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
