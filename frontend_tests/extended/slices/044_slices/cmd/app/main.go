package main

func main() {
	values := []int{0, 6, 12}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
