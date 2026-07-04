package main

type count int
type text string

func main() {
	v := count(16)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 21 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
