package main

type count int
type text string

func main() {
	v := count(36)
	s := text("PASS\n")
	if int(v)+len(string(s)) == 41 {
		print(string(s))
		return
	}
	print("FAIL\n")
}
