package main

func main() {
	values := []int{8, 10, 8}
	values = append(values[1:2], 20)
	if len(values) == 2 && values[0]+values[1] == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
