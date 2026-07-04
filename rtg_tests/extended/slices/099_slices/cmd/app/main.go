package main

func main() {
	values := []int{0, 9, 16}
	values = append(values[1:2], 7)
	if len(values) == 2 && values[0]+values[1] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
