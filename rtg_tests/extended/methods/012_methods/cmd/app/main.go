package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 12
	if int(c.add(12)) == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
