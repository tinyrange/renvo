package main

func main() {
	values := []int{0, 12, 13}
	values = append(values[1:2], 14)
	if len(values) == 2 && values[0]+values[1] == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
