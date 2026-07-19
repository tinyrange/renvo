package main

type item struct {
	name string
	code int
}

func appMain() int {
	items := []item{{name: "b", code: 2}, {name: "a", code: 1}, {}}
	j := 1
	value := items[j]
	items[j+1] = items[j]
	items[j] = items[j-1]
	items[j-1] = value
	if items[0].name != "a" || items[0].code != 1 {
		print("FAIL\n")
		return 1
	}
	if items[1].name != "b" || items[1].code != 2 {
		print("FAIL\n")
		return 1
	}
	if items[2].name != "a" || items[2].code != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
