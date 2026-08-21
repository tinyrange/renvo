package main

type aggregateArgumentResultPair struct {
	first  int
	second int
}

func aggregateArgumentResultAdd(left aggregateArgumentResultPair, right aggregateArgumentResultPair) aggregateArgumentResultPair {
	return aggregateArgumentResultPair{
		first:  left.first + right.first,
		second: left.second + right.second,
	}
}

func appMain(args []string) int {
	result := aggregateArgumentResultAdd(
		aggregateArgumentResultPair{first: 1, second: 2},
		aggregateArgumentResultPair{first: 10, second: 20},
	)
	if result.first != 11 || result.second != 22 {
		print("aggregate argument/result failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
