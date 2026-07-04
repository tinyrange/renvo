package main

func main() {
	values := []int{0, 1, 2}
	values = append(values[1:2], 3)
	if len(values) == 2 && values[0]+values[1] == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
