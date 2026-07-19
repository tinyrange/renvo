package main

func main() {
	values := []int{2, 4, 2}
	values = append(values[1:2], 14)
	if len(values) == 2 && values[0]+values[1] == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
