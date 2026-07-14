package main

type counter int

func (c counter) add(v int) counter {
	return c + counter(v)
}

func main() {
	var c counter = 1
	corpusOK := int(c.add(1)) == 2
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
