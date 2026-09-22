package main

type Pair struct{ X, Y int }
type Alias = Pair
type Outer struct{ Value *Pair }
type Embedded struct {
	Pair
	Label string
}
type EmbeddedPointer struct{ *Pair }

var zero = Pair{}
var keyed = Pair{Y: 4, X: 3}

func main() {
	positional := Alias{1, 2}
	nested := Outer{&Pair{X: 5, Y: 6}}
	if zero.X != 0 || zero.Y != 0 || keyed.X != 3 || keyed.Y != 4 || positional.X != 1 || positional.Y != 2 || nested.Value.X != 5 || nested.Value.Y != 6 {
		panic("struct literal values")
	}
	embedded := Embedded{Pair: Pair{X: 7, Y: 8}, Label: "ok"}
	pointer := EmbeddedPointer{Pair: &Pair{X: 9, Y: 10}}
	if embedded.Pair.X != 7 || embedded.Pair.Y != 8 || embedded.Label != "ok" || pointer.Pair.X != 9 || pointer.Pair.Y != 10 {
		panic("embedded struct literal values")
	}
	print("PASS\n")
}
