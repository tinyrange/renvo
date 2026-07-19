package main

func main() {
	values := []int{9, 7, 14}
	values = append(values[1:2], 5)
	if len(values) == 2 && values[0]+values[1] == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
