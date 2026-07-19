package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 25
	if int(c.add(8)) == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
