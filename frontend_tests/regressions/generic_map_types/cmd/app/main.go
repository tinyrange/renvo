package main

type Named (map[string]int)
type Alias = (map[string]int)

func New[K comparable, V any]() map[K]V { return make(map[K]V) }
func Nested[K comparable, V any]() map[K]V {
	return make((map[K]V), 2)
}
func Nil[K comparable, V any]() map[K]V { return (map[K]V)(nil) }
func Pointer[K comparable, V any]() *map[K]V {
	return new((map[K]V))
}
func Identity[T any](v T) T { return v }

func main() {
	m := New[string, int]()
	m["value"] = 42
	if m == nil || len(m) != 1 || m["value"] != 42 {
		panic("map creation")
	}
	nested := Nested[int, string]()
	nested[42] = "value"
	if nested[42] != "value" || Nil[int, string]() != nil {
		panic("parenthesized map type")
	}
	pointer := Pointer[string, int]()
	if pointer == nil || *pointer != nil {
		panic("pointer to nil map")
	}
	*pointer = m
	if (*pointer)["value"] != 42 {
		panic("map pointer assignment")
	}
	named := make((Named), 2)
	alias := make((Alias))
	named["value"], alias["value"] = 17, 25
	if named["value"]+alias["value"] != 42 {
		panic("parenthesized declared map types")
	}
	var boxed any = Identity(named)
	if _, ok := boxed.(Named); !ok {
		panic("named map identity")
	}
	if _, ok := boxed.(map[string]int); ok {
		panic("named map acquired anonymous identity")
	}
	boxed = Identity(alias)
	if _, ok := boxed.(Named); ok {
		panic("map alias acquired named identity")
	}
	if _, ok := boxed.(map[string]int); !ok {
		panic("map alias identity")
	}
	print("PASS\n")
}
