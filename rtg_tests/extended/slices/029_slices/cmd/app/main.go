package main

func main() {
	values := []int{7, 4, 14}
	values = append(values[1:2], 13)
	if len(values) == 2 && values[0]+values[1] == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
