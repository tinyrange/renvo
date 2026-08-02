package main

type rtgStringList struct {
	items []string
}

func appMain(args []string) int {
	var list rtgStringList
	for _, item := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
		list.items = append(list.items, item)
	}
	if len(list.items) == 10 && list.items[0] == "0" && list.items[9] == "9" {
		print("PASS\n")
		return 0
	} else {
		print("FAIL\n")
		return 1
	}
}
