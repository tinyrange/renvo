package main

func main() {
	values := []int{6, 3, 13}
	values = append(values[1:2], 12)
	if len(values) == 2 && values[0]+values[1] == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
