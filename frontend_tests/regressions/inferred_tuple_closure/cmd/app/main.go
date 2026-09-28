package main

type value struct {
	n int
	s string
}
type entry struct {
	key value
	val value
}

func makeEntries(base int) []entry {
	calls := 0
	convert := func(n int) (value, error) {
		calls++
		return value{base + n, "converted"}, nil
	}
	var entries []entry
	for i := 0; i < 2; i++ {
		key, err := convert(i)
		if err != nil {
			panic("key")
		}
		v, err := convert(i + 10)
		if err != nil {
			panic("value")
		}
		entries = append(entries, entry{key, v})
	}
	if calls != 4 {
		panic("calls")
	}
	return entries
}

func main() {
	items := makeEntries(20)
	if len(items) != 2 || items[0].key.n != 20 || items[1].val.n != 31 || items[1].key.s != "converted" {
		panic("tuple capture")
	}
	println("PASS")
}
