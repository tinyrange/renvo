package main

type info struct {
	text string
	pos  int
}

type entry struct {
	key  string
	info info
}

func pick(entries []entry, key string) info {
	for i := 0; i < len(entries); i++ {
		item := entries[i]
		if item.key == key {
			return item.info
		}
	}
	return info{text: "FAIL\n", pos: -1}
}

func appMain() int {
	entries := []entry{{key: "ok", info: info{text: "PASS\n", pos: 1}}}
	got := pick(entries, "ok")
	if got.text == "PASS\n" && got.pos == 1 {
		print(got.text)
		return 0
	}
	print("FAIL\n")
	return 1
}
