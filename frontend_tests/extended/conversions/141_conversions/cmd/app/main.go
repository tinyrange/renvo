package main

type count int
type text string

func main() {
	v := count(30)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 35 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
