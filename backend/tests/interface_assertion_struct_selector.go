package main

type assertionSelectorRecord struct {
	pair [2]int64
}

func appMain(args []string) int {
	var dynamic any = assertionSelectorRecord{pair: [2]int64{11, 17}}
	total := int64(3)
	total += dynamic.(assertionSelectorRecord).pair[1]
	if total != 20 {
		print("FAIL: struct selector after interface assertion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
