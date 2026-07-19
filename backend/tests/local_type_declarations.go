package main

func localTypeInt() int {
	type number int
	type box struct {
		value number
	}
	item := box{value: number(41)}
	return int(item.value)
}

func localTypeByte() int {
	type number byte
	type box struct {
		value number
	}
	raw := 298
	item := box{value: number(raw)}
	return int(item.value)
}

func appMain() int {
	if localTypeInt() != 41 || localTypeByte() != 42 {
		print("local type declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
