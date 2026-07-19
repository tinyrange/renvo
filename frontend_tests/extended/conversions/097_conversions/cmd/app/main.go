package main

type count int
type text string

func main() {
	v := count(23)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 28 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
