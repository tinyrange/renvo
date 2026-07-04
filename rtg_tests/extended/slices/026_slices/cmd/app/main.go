package main

func main() {
	values := []int{4, 1, 11}
	values = append(values[1:2], 10)
	if len(values) == 2 && values[0]+values[1] == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
