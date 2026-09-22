package main

type Pair struct{ X, Y int }
type Alias = Pair
type Outer struct{ Value *Pair }

var zero = Pair{}
var keyed = Pair{Y: 4, X: 3}

func main() {
	positional := Alias{1, 2}
	nested := Outer{&Pair{X: 5, Y: 6}}
	if zero.X != 0 || zero.Y != 0 || keyed.X != 3 || keyed.Y != 4 || positional.X != 1 || positional.Y != 2 || nested.Value.X != 5 || nested.Value.Y != 6 {
		panic("struct literal values")
	}
	print("PASS\n")
}
