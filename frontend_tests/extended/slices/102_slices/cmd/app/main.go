package main

func main() {
	values := []int{3, 12, 2}
	values = append(values[1:2], 10)
	if len(values) == 2 && values[0]+values[1] == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
