package main

func main() {
	values := [3]int{6, 3, 9}
	total := values[0] + values[1]*2 + values[2]*3
	if total == 39 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
