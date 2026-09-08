package main

func scan(name []uint16) int {
	i := 0
	for i < len(name) && name[i] != 0 && name[i] != 0xffff {
		i++
	}
	return i
}

func appMain() int {
	var name [40]uint16
	for i := range name {
		name[i] = 0xffff
	}
	for i := 0; i < 26; i++ {
		name[i] = 'a'
	}
	i := 0
	for i < len(name) && name[i] != 0xffff {
		i++
	}
	if i != 26 || scan(name[:]) != 26 {
		print("FAIL sentinel\n")
		return 1
	}
	if name[i] != 65535 || name[i] < 32768 {
		print("FAIL unsigned\n")
		return 1
	}
	var signed [1]int16
	signed[0] = -1
	if signed[0] != -1 {
		print("FAIL signed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
