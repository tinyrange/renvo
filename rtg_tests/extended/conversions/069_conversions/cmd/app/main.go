package main

type count int
type text string

func main() {
	v := count(32)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 37 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
