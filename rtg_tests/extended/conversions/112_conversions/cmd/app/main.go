package main

type count int
type text string

func main() {
	v := count(1)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 6 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
