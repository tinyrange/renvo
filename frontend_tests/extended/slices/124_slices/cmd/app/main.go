package main

func main() {
	values := []int{3, 8, 7}
	values = append(values[1:2], 13)
	if len(values) == 2 && values[0]+values[1] == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
