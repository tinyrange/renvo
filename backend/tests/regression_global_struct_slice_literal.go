package main

type renvoGlobalStructSliceItem struct {
	value int
}

type renvoGlobalStructSliceItems []renvoGlobalStructSliceItem

var renvoGlobalStructSliceValues renvoGlobalStructSliceItems = renvoGlobalStructSliceItems{renvoGlobalStructSliceItem{value: 4}, renvoGlobalStructSliceItem{6}}

func appMain(args []string) int {
	if len(renvoGlobalStructSliceValues) != 2 {
		print("global struct slice literal length failed\n")
		return 1
	}
	if renvoGlobalStructSliceValues[0].value+renvoGlobalStructSliceValues[1].value != 10 {
		print("global struct slice literal value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
