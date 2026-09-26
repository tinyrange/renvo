package main

type item struct{ Name string }
type record struct {
	Lookup map[string]item `json:"lookup"`
	Fixed  [2]int
}

func main() {
	var value record
	value.Lookup = map[string]item{"a": {Name: "mapped"}}
	if value.Lookup["a"].Name != "mapped" || value.Fixed[0] != 0 {
		panic("tagged field type")
	}
	print("PASS\n")
}
