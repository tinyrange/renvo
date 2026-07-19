package main

type count int
type text string

func main() {
	v := count(1)
	s := text("PASS\n")
	corpusOK := int(v)+len(string(s)) == 6
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print(string(s))

}
