package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 19
	if int(c.add(13)) == 32 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
