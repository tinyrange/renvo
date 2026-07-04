package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 14
	if int(c.add(14)) == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
