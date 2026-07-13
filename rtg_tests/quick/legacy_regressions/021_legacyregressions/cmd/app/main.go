package main

func main() {
	var a = [2]int{1, 2}
	if len(a) == 2 && a[0] == 1 && a[1] == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
