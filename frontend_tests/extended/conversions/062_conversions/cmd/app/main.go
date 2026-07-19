package main

type count int
type text string

func main() {
	v := count(25)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 30 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
