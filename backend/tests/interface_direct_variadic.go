package main

type directVariadicValue int

func (v directVariadicValue) Sum(xs ...int) int {
	n := int(v)
	for i := 0; i < len(xs); i++ {
		n += xs[i]
	}
	return n
}

type directVariadicContainer struct {
	padding int
	directVariadicValue
}

func appMain(args []string) int {
	var value interface{ Sum(...int) int } = directVariadicValue(10)
	if value.Sum(1, 2) != 13 || value.Sum() != 10 || value.Sum([]int{3, 4}...) != 17 {
		return 1
	}
	value = directVariadicContainer{padding: 99, directVariadicValue: 20}
	if value.Sum(5, 6, 7) != 38 || value.Sum() != 20 || value.Sum([]int{8, 9}...) != 37 {
		return 2
	}
	print("PASS\n")
	return 0
}
