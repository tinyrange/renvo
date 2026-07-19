package main

func main() {
	values := []int{6, 5, 2}
	values = append(values[1:2], 20)
	if len(values) == 2 && values[0]+values[1] == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
