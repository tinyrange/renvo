package main

func main() {
	values := []int{9, 8, 5}
	values = append(values[1:2], 4)
	if len(values) == 2 && values[0]+values[1] == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
