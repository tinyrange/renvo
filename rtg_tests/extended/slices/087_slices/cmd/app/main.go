package main

func main() {
	values := []int{10, 10, 4}
	values = append(values[1:2], 14)
	if len(values) == 2 && values[0]+values[1] == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
