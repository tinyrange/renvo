package main

func main() {
	values := []int{2, 1, 15}
	values = append(values[1:2], 16)
	if len(values) == 2 && values[0]+values[1] == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
