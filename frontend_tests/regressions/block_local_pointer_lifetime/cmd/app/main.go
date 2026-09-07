package main

type record struct {
	text string
	n    int
}
type holder struct{ value *record }

func number(v record) int { return v.n }
func main() {
	h := holder{}
	{
		local := record{"kept", 41}
		h.value = &local
		local.n++
	}
	{
		overwrite := record{"overwritten", 3}
		if number(overwrite) != 3 {
			panic("overwrite")
		}
	}
	if h.value.text != "kept" || h.value.n != 42 {
		panic("block lifetime")
	}
	println("PASS")
}
