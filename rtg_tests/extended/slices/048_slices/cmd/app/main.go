package main

func main() {
	values := []int{4, 10, 16}
	values = append(values[1:2], 13)
	if len(values) == 2 && values[0]+values[1] == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
