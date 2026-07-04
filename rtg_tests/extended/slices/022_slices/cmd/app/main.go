package main

func main() {
	values := []int{0, 10, 7}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
