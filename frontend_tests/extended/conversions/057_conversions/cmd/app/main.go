package main

type count int
type text string

func main() {
	v := count(20)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 25 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
