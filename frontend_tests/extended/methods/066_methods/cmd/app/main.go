package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 4
	if int(c.add(15)) == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
