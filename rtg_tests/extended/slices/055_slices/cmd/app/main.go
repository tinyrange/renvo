package main

func main() {
	values := []int{0, 4, 6}
	values = append(values[1:2], 20)
	if len(values) == 2 && values[0]+values[1] == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
