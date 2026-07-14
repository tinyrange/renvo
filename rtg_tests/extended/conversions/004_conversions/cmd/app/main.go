package main

type count int
type text string

func main() {
	v := count(4)
	s := text("PASS\n")
	corpusOK := false
	if int(v)+len(string(s)) == 9 {
		corpusOK = true
	}
	if corpusOK {
		print(string(s))
		return
	}

	print("FAIL\n")
}
