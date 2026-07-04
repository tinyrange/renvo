package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 5
	if int(c.add(13)) == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
