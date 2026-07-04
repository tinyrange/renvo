package main

func main() {
	values := [3]int{9, 9, 9}
	total := values[0] + values[1]*2 + values[2]*3
	if total == 54 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
