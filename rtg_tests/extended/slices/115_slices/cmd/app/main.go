package main

func main() {
	values := []int{5, 12, 15}
	values = append(values[1:2], 4)
	if len(values) == 2 && values[0]+values[1] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
