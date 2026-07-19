package main

type renvoNamedSliceAppendItem struct {
	name string
}

type renvoNamedSliceAppendItems []renvoNamedSliceAppendItem

func renvoNamedSliceAppendCopy(out renvoNamedSliceAppendItems, values renvoNamedSliceAppendItems) renvoNamedSliceAppendItems {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func appMain() int {
	values := renvoNamedSliceAppendItems{renvoNamedSliceAppendItem{name: "PASS\n"}}
	out := renvoNamedSliceAppendCopy(renvoNamedSliceAppendItems{}, values)
	if len(out) == 1 {
		if out[0].name == "PASS\n" {
			print(out[0].name)
			return 0
		}
	}
	print("FAIL\n")
	return 1
}
