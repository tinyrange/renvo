package main

type renvoAppendPackedValue [3]byte

func appMain(args []string) int {
	destination := make([]renvoAppendPackedValue, 6)
	for i := 0; i < len(destination); i++ {
		destination[i] = renvoAppendPackedValue{9, 9, 9}
	}
	source := []renvoAppendPackedValue{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}
	result := append(destination[:1], source[:2]...)
	if len(result) != 3 || result[1][0] != 1 || result[1][1] != 2 || result[1][2] != 3 || result[2][0] != 4 || result[2][1] != 5 || result[2][2] != 6 || destination[4][0] != 9 || destination[4][1] != 9 || destination[4][2] != 9 {
		return 1
	}
	print("PASS\n")
	return 0
}
