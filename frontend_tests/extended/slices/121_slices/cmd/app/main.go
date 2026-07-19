package main

func main() {
	values := []int{0, 5, 4}
	values = append(values[1:2], 10)
	if len(values) == 2 && values[0]+values[1] == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
