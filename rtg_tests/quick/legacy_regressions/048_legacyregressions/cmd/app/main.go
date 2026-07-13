package main

func main() {
	x := 4
	p := &x
	(*p)++
	if x == 5 && *p == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
