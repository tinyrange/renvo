package main

func main() {
	values := []int{10, 1, 16}
	values = append(values[1:2], 11)
	if len(values) == 2 && values[0]+values[1] == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
