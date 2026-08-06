package main

func main() {
	values := []int64{9, -4, 17}
	array := [...]int64{8, 6, 4}
	mapping := map[string]int64{"first": 20, "second": -3}
	text := "go"
	mapMaximum := max(mapping["first"], mapping["second"])
	clear(values[1:2])
	clear(array[1:])
	clear(mapping)
	if min(values[0], values[1]) != 0 || array[0] != 8 || array[1] != 0 || array[2] != 0 || mapMaximum != 20 || min(text[0], text[1]) != 'g' || len(mapping) != 0 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
