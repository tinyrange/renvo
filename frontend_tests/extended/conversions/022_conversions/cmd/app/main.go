package main

type count int
type text string

func main() {
	v := count(22)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 27 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
