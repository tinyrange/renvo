package main

func main() {
	values := []int{7, 3, 6}
	values = append(values[1:2], 14)
	if len(values) == 2 && values[0]+values[1] == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
