package main

type item struct {
	n    int
	text string
}

func makeItems() []*item { return []*item{{n: 3, text: "first"}, {n: 7}} }

func appMain(args []string) int {
	items := makeItems()
	array := [2]*item{{n: 11}, {n: 19}}
	if items[0] == items[1] || items[0].n != 3 || items[0].text != "first" || items[1].n != 7 || items[1].text != "" || array[0].n != 11 || array[1].n != 19 {
		return 1
	}
	items[0].n = 42
	if items[1].n != 7 {
		return 2
	}
	print("PASS\n")
	return 0
}
