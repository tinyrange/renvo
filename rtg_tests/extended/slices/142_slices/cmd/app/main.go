package main

func main() {
	values := []int{10, 13, 8}
	values = append(values[1:2], 12)
	if len(values) == 2 && values[0]+values[1] == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
