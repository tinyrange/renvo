package main

type count int
type text string

func main() {
	v := count(9)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 14 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
