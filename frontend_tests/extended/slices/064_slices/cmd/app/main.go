package main

func main() {
	values := []int{9, 13, 15}
	values = append(values[1:2], 10)
	if len(values) == 2 && values[0]+values[1] == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
