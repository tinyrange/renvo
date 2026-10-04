package main

const arrayTargetWordLength = int(^uint(0) >> 2)

type arrayTargetWordBlock [arrayTargetWordLength]struct{}

func appMain() int {
	var value *arrayTargetWordBlock
	if Sizeof(int(0)) == 8 {
		if len(value) != (1<<62)-1 {
			return 1
		}
	} else if len(value) != (1<<30)-1 {
		return 2
	}
	print("PASS\n")
	return 0
}
