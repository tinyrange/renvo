package main

func main() {
	values := []int{5, 2, 12}
	values = append(values[1:2], 11)
	if len(values) == 2 && values[0]+values[1] == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
