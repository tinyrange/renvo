package main

func main() {
	values := []int{10, 11, 12}
	values = append(values[1:2], 13)
	if len(values) == 2 && values[0]+values[1] == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
