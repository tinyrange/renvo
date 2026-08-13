package main

func main() {
	const (
		blah  = 20
		blah2 = 22
	)
	if blah+blah2 != 42 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
