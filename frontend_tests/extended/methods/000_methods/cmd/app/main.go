package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 0
	if int(c.add(0)) == 0 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
