package main

func main() {
	values := [3]int{7, 5, 3}
	total := values[0] + values[1]*2 + values[2]*3
	if total == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
