package main

func main() {
	values := []int{10, 7, 17}
	values = append(values[1:2], 16)
	if len(values) == 2 && values[0]+values[1] == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
