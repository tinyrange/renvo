package main

func main() {
	values := []int{3, 4, 12}
	values = append(values[1:2], 16)
	if len(values) == 2 && values[0]+values[1] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
