package main

func main() {
	values := []int{6, 6, 17}
	values = append(values[1:2], 10)
	if len(values) == 2 && values[0]+values[1] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
