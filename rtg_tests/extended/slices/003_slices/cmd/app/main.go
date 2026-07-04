package main

func main() {
	values := []int{3, 4, 5}
	values = append(values[1:2], 6)
	if len(values) == 2 && values[0]+values[1] == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
