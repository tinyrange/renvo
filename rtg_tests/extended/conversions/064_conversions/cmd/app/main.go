package main

type count int
type text string

func main() {
	v := count(27)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 32 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
