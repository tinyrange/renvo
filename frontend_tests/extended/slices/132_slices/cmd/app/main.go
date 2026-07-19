package main

func main() {
	values := []int{0, 3, 15}
	values = append(values[1:2], 21)
	if len(values) == 2 && values[0]+values[1] == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
