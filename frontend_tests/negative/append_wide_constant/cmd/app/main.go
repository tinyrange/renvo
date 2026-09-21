package main

const Huge = 1 << 100

func main() {
	_ = append([]int64{}, Huge)
}
