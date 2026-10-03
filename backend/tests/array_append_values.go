package main

type ArrayAppendValue [2]int

var arrayAppendCalls int

func arrayAppendPair(first int, second int) ArrayAppendValue {
	arrayAppendCalls++
	return ArrayAppendValue{first, second}
}

func appMain(args []string) int {
	values := make([]ArrayAppendValue, 0, 1)
	values = append(values, arrayAppendPair(17, 42))
	values = append(values, ArrayAppendValue{51, 73})
	more := ArrayAppendValue{91, 101}
	values = append(values, more, arrayAppendPair(111, 121))
	if len(values) != 4 || values[0][0] != 17 || values[0][1] != 42 || values[1][0] != 51 || values[1][1] != 73 || values[2][0] != 91 || values[2][1] != 101 || values[3][0] != 111 || values[3][1] != 121 || arrayAppendCalls != 2 {
		panic("array append value or evaluation")
	}
	print("PASS\n")
	return 0
}
