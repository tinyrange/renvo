package main

type mapFieldBox struct {
	values map[string]int
}

func appMain() int {
	values := map[string]int{"zero": 0, "one": 1}
	zero, zeroOK := values["zero"]
	missing, missingOK := values["missing"]
	box := mapFieldBox{values: values}
	box.values["two"] = 2
	if zero == 0 && zeroOK && missing == 0 && !missingOK && box.values["one"] == 1 && box.values["two"] == 2 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
