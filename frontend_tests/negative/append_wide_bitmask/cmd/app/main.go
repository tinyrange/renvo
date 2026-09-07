package main

func main() {
	_ = append([]int8{}, ((1<<100)|128)&255)
}
