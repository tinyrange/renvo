package main

func main() {
	values := []int{5, 6, 14}
	values = append(values[1:2], 18)
	if len(values) == 2 && values[0]+values[1] == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
