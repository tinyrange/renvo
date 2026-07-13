package main

func main() {
	var s []int
	if len(s) == 0 && cap(s) == 0 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
