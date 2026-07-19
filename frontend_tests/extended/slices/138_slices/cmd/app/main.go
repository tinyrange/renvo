package main

func main() {
	values := []int{6, 9, 4}
	values = append(values[1:2], 8)
	if len(values) == 2 && values[0]+values[1] == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
