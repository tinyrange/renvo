package main

func main() {
	values := []int{9, 3, 2}
	values = append(values[1:2], 8)
	if len(values) == 2 && values[0]+values[1] == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
