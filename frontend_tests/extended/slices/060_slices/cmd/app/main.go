package main

func main() {
	values := []int{5, 9, 11}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
