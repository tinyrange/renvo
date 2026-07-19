package main

type item struct {
	kind  int
	text  string
	start int32
	end   int32
}

func makeItem(kind int, text string, start int, end int) item {
	return item{kind: kind, text: text, start: int32(start), end: int32(end)}
}

func appMain(args []string, env []string) int {
	var items []item
	items = append(items, makeItem(7, "ok", 3, 5))
	if len(items) != 1 {
		return 1
	}
	if items[0].kind != 7 {
		return 1
	}
	if items[0].text != "ok" {
		return 1
	}
	if items[0].start != 3 {
		return 1
	}
	if items[0].end != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
