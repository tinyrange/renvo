package main

func main() {
	values := []int{7, 9, 7}
	values = append(values[1:2], 19)
	if len(values) == 2 && values[0]+values[1] == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
