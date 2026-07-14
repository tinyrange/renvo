package main

type UnsupportedMapField struct {
	Field map[string]struct{ A int }
}
func main() { _ = UnsupportedMapField{} }
