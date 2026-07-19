package main

type count int
type text string

func main() {
	v := count(2)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 7 {
		print(string(s))
		return
	} else {

		print("FAIL\n")
	}
}
