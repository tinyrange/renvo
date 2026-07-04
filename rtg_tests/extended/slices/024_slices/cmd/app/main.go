package main

func main() {
	values := []int{2, 12, 9}
	values = append(values[1:2], 8)
	if len(values) == 2 && values[0]+values[1] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
