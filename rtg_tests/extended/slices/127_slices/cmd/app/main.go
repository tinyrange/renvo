package main

func main() {
	values := []int{6, 11, 10}
	values = append(values[1:2], 16)
	if len(values) == 2 && values[0]+values[1] == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
