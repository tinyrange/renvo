package main

func main() {
	values := []int{1, 11, 8}
	values = append(values[1:2], 7)
	if len(values) == 2 && values[0]+values[1] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
