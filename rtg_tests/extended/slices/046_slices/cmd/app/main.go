package main

func main() {
	values := []int{2, 8, 14}
	values = append(values[1:2], 11)
	if len(values) == 2 && values[0]+values[1] == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
