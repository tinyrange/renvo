package main

type count int
type text string

func main() {
	v := count(15)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 20 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
