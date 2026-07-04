package main

func main() {
	values := []int{3, 5, 3}
	values = append(values[1:2], 15)
	if len(values) == 2 && values[0]+values[1] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
