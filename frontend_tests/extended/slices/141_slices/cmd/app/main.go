package main

func main() {
	values := []int{9, 12, 7}
	values = append(values[1:2], 11)
	if len(values) == 2 && values[0]+values[1] == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
