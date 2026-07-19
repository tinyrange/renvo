package main

func main() {
	values := []int{6, 7, 8}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
