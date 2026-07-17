package main

type callResultSelectorSignature struct {
	result int
	params []int
}

func callResultSelectorBuild() callResultSelectorSignature {
	return callResultSelectorSignature{result: 99, params: []int{1, 2, 3}}
}

func appMain(args []string) int {
	params := callResultSelectorBuild().params
	total := 0
	for i := 0; i < len(params); i++ {
		total += params[i]
	}
	if total != 6 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
