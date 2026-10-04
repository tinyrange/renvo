package main

import "example.com/genericdotidentity/callback"

func main() {
	Unique := func() int { return 31 }
	type Type int
	Counter := 37
	Width := 41
	if callback.Id(callback.Unique()) != 17 || ReadDot() != 17 || Unique() != 31 {
		panic(1)
	}
	if ReadGeneric(callback.Type{N: 43}).N != 43 || Type(31) != 31 {
		panic(2)
	}
	if Initialized != 19 || Counter != 37 || Width != 41 {
		panic(3)
	}
	v := IdentityType(callback.Type{N: 47})
	if v.N != 47 || ReadCollections() != 59 {
		panic(4)
	}
	if ReadCallback() != 17 {
		panic(5)
	}
	if ReadFields() != 6 {
		panic(6)
	}
	print("PASS\n")
}
