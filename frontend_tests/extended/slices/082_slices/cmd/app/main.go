package main

func main() {
	values := []int{5, 5, 16}
	values = append(values[1:2], 9)
	if len(values) == 2 && values[0]+values[1] == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
