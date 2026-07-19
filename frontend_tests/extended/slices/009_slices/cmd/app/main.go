package main

func main() {
	values := []int{9, 10, 11}
	values = append(values[1:2], 12)
	if len(values) == 2 && values[0]+values[1] == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
