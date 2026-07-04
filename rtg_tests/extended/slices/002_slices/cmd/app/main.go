package main

func main() {
	values := []int{2, 3, 4}
	values = append(values[1:2], 5)
	if len(values) == 2 && values[0]+values[1] == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
