package main

func main() {
	values := []int{3, 3, 14}
	values = append(values[1:2], 7)
	if len(values) == 2 && values[0]+values[1] == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
