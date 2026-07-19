package main

type multiAppendPair struct {
	left  int
	right int
}

var multiAppendOrder int

func nextMultiAppendValue() int {
	multiAppendOrder++
	return multiAppendOrder
}

func nextMultiAppendPair(value int) multiAppendPair {
	multiAppendOrder++
	return multiAppendPair{left: value, right: multiAppendOrder}
}

func appMain(args []string) int {
	values := []byte{1}
	values = append(values, byte(2), byte(3), byte(4))
	if len(values) != 4 || values[0] != 1 || values[1] != 2 || values[2] != 3 || values[3] != 4 {
		print("FAIL bytes\n")
		return 1
	}

	words := []string{"first"}
	words = append(words, "second", "third")
	if len(words) != 3 || words[1] != "second" || words[2] != "third" {
		print("FAIL strings\n")
		return 1
	}

	numbers := []int{9}
	numbers = append(numbers, len(numbers), len(numbers))
	if len(numbers) != 3 || numbers[1] != 1 || numbers[2] != 1 {
		print("FAIL pre-mutation evaluation\n")
		return 1
	}

	multiAppendOrder = 0
	ordered := []int{}
	ordered = append(ordered, nextMultiAppendValue(), nextMultiAppendValue(), nextMultiAppendValue())
	if len(ordered) != 3 || ordered[0] != 1 || ordered[1] != 2 || ordered[2] != 3 || multiAppendOrder != 3 {
		print("FAIL evaluation order\n")
		return 1
	}

	multiAppendOrder = 0
	pairs := []multiAppendPair{}
	pairs = append(pairs, multiAppendPair{left: 1, right: 2}, nextMultiAppendPair(3), nextMultiAppendPair(4))
	if len(pairs) != 3 || pairs[0].right != 2 || pairs[1].left != 3 || pairs[1].right != 1 || pairs[2].left != 4 || pairs[2].right != 2 {
		print("FAIL structs\n")
		return 1
	}

	first, second := 7, 8
	pointers := []*int{}
	pointers = append(pointers, &first, &second)
	if len(pointers) != 2 || *pointers[0] != 7 || *pointers[1] != 8 {
		print("FAIL pointers\n")
		return 1
	}

	narrow := []int8{}
	narrow = append(narrow, int8(-1), int8(-128), int8(127))
	if len(narrow) != 3 || int(narrow[0]) != -1 || int(narrow[1]) != -128 || int(narrow[2]) != 127 {
		print("FAIL int8\n")
		return 1
	}

	print("PASS\n")
	return 0
}
