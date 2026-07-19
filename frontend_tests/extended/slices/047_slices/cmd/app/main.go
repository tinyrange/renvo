package main

func main() {
	values := []int{3, 9, 15}
	values = append(values[1:2], 12)
	if len(values) == 2 && values[0]+values[1] == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
