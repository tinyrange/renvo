package main

func main() {
	values := []int{1, 13, 14}
	values = append(values[1:2], 15)
	if len(values) == 2 && values[0]+values[1] == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
