package main

func main() {
	values := [3]int{8, 7, 6}
	total := values[0] + values[1]*2 + values[2]*3
	if total == 40 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
