package main

func main() {
	values := []int{7, 5, 12}
	values = append(values[1:2], 3)
	if len(values) == 2 && values[0]+values[1] == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
