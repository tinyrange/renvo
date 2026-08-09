package main

var c89Seed = 11

func c89Mix(left int, right int) int {
	return (left*3 + right) ^ 5
}

func c89Sum(limit int) int {
	total := 0
	for i := 0; i <= limit; i++ {
		if i == 3 {
			continue
		}
		total += i
		if total > 50 {
			break
		}
	}
	return total
}

func c89Eight(a, b, c, d, e, f, g, h int) int {
	return a + b + c + d + e + f + g + h
}

type c89Block struct {
	left  int
	right int
	bytes [3]byte
}

func appMain(args []string) int {
	if len(args) != 1 || len(args[0]) == 0 {
		return 13
	}
	if c89Seed != 11 || c89Mix(7, 3) != 29 || c89Sum(10) != 52 {
		return 1
	}
	if -7/3 != -2 || -7%3 != -1 || -16>>2 != -4 {
		return 2
	}
	maximum := 2147483647
	if maximum+1 != -2147483648 {
		return 3
	}
	if !(-2147483648 < 1) || !(2147483647 > -1) {
		return 4
	}
	var data [8]byte
	for i := 0; i < len(data); i++ {
		data[i] = byte(i*7 + 1)
	}
	if data[0] != 1 || data[3] != 22 || data[7] != 50 {
		return 5
	}
	value := 1
	value <<= 31
	if value != -2147483648 {
		return 6
	}
	value >>= 32
	if value != -1 {
		return 7
	}
	var signed8 int8 = -7
	var signed16 int16 = -300
	var words [4]uint16
	words[2] = 65530
	if int(signed8) != -7 {
		return 81
	}
	if int(signed16) != -300 {
		return 82
	}
	if int(words[2]) != 65530 {
		return 83
	}
	pointer := &data[3]
	*pointer = 99
	if data[3] != 99 {
		return 9
	}
	left := c89Block{left: 17, right: -9, bytes: [3]byte{4, 5, 6}}
	right := left
	if right.left != 17 || right.right != -9 || right.bytes[2] != 6 {
		return 10
	}
	if c89Eight(1, 2, 3, 4, 5, 6, 7, 8) != 36 {
		return 11
	}
	minimum := -2147483648
	maximumUnsigned := uint32(maximum+1)*2 - 1
	if maximumUnsigned <= uint32(1) {
		return 121
	}
	if minimum/-1 != minimum {
		return 122
	}
	overlap := [6]byte{1, 2, 3, 4, 5, 6}
	copy(overlap[1:], overlap[:5])
	if overlap != [6]byte{1, 1, 2, 3, 4, 5} {
		return 14
	}
	return 0
}
