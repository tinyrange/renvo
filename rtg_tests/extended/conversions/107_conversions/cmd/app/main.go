package main

type count int
type text string

func main() {
	v := count(33)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 38 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
