package main

func main() {
	values := []int{8, 8, 2}
	values = append(values[1:2], 12)
	if len(values) == 2 && values[0]+values[1] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
