package main

func main() {
	values := []int{0, 13, 11}
	values = append(values[1:2], 4)
	if len(values) == 2 && values[0]+values[1] == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
