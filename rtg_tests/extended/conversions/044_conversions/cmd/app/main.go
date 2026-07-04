package main

type count int
type text string

func main() {
	v := count(7)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 12 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
