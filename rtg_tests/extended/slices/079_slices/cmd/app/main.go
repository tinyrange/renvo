package main

func main() {
	values := []int{2, 2, 13}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
