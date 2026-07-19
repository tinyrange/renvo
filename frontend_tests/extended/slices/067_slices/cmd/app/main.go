package main

func main() {
	values := []int{1, 3, 18}
	values = append(values[1:2], 13)
	if len(values) == 2 && values[0]+values[1] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
