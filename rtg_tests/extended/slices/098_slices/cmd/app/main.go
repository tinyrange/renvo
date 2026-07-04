package main

func main() {
	values := []int{10, 8, 15}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
