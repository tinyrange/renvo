package main

func main() {
	values := []int{6, 13, 16}
	values = append(values[1:2], 5)
	if len(values) == 2 && values[0]+values[1] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
