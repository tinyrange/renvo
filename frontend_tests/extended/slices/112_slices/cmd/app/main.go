package main

func main() {
	values := []int{2, 9, 12}
	values = append(values[1:2], 20)
	if len(values) == 2 && values[0]+values[1] == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
