package main

func main() {
	values := []int{1, 6, 5}
	values = append(values[1:2], 11)
	if len(values) == 2 && values[0]+values[1] == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
