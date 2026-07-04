package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 21
	if int(c.add(1)) == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
