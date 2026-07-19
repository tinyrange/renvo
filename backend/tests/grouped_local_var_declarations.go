package main

type groupedLocalPair struct {
	left  int
	right int
}

func groupedLocalBump(value *int) int {
	*value = *value + 1
	return *value
}

func appMain(args []string) int {
	var first, second int
	second = 7
	if first != 0 || second != 7 {
		print("FAIL zero values\n")
		return 1
	}

	var left, right string = "left", "right"
	if left != "left" || right != "right" {
		print("FAIL strings\n")
		return 1
	}

	var one, two groupedLocalPair
	two.right = 9
	if one.left != 0 || one.right != 0 || two.left != 0 || two.right != 9 {
		print("FAIL structs\n")
		return 1
	}

	outer := 5
	{
		var outer, next int = outer + 1, outer + 2
		if outer != 6 || next != 7 {
			print("FAIL initializer scope\n")
			return 1
		}
	}
	if outer != 5 {
		print("FAIL outer scope\n")
		return 1
	}

	calls := 0
	var _, last int = groupedLocalBump(&calls), groupedLocalBump(&calls)
	if calls != 2 || last != 2 {
		print("FAIL initializer order\n")
		return 1
	}

	var firstArray, secondArray [2]int = [2]int{1, 2}, [2]int{3, 4}
	if firstArray[0] != 1 || firstArray[1] != 2 || secondArray[0] != 3 || secondArray[1] != 4 {
		print("FAIL arrays\n")
		return 1
	}

	var firstSlice, secondSlice []int = []int{8}, []int{9}
	if len(firstSlice) != 1 || firstSlice[0] != 8 || len(secondSlice) != 1 || secondSlice[0] != 9 {
		print("FAIL slices\n")
		return 1
	}

	print("PASS\n")
	return 0
}
