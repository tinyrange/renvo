package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 18
	if int(c.add(12)) == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
