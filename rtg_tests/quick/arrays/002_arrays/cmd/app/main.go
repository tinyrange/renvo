package main

func main() {
	values := [3]int{3, 6, 9}
	total := values[0] + values[1]*2 + values[2]*3
	if total == 42 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
