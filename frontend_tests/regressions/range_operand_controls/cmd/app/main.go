package main

type Count int

func main() {
	total := 0
	for range Count(3) {
		total++
	}
	for range "abc" {
		total++
	}
	for range []int{1, 2} {
		total++
	}
	for range &[2]int{} {
		total++
	}
	for range map[int]int{1: 2} {
		total++
	}
	value := false
	{
		value := 2
		for range value {
			total++
		}
	}
	for range struct{ values []int }{values: []int{1}}.values {
		total++
	}
	if value || total != 14 {
		panic("range operands")
	}
	println("PASS")
}
