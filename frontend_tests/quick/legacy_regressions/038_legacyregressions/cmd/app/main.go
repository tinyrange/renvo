package main

func main() {
	a, b := 1, 2
	p := &a
	*p, b = b, *p
	if a == 2 && b == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
