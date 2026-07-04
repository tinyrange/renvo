package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 17
	if int(c.add(8)) == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
