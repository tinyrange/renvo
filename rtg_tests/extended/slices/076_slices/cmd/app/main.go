package main

func main() {
	values := []int{10, 12, 10}
	values = append(values[1:2], 3)
	if len(values) == 2 && values[0]+values[1] == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
