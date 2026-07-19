package main

func main() {
	values := []int{3, 2, 16}
	values = append(values[1:2], 17)
	if len(values) == 2 && values[0]+values[1] == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
