package main

type count int
type text string

func main() {
	v := count(3)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 8 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
