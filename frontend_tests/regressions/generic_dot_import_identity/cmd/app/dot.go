package main

import . "example.com/genericdotidentity/callback"

type Alias = Type
type Numbers [Width]int

var Initialized = Counter

func ReadDot() int              { return Unique() }
func ReadGeneric[T any](v T) T  { return Id(v) }
func IdentityType(v Type) Alias { return Id(v) }
func ReadCollections() int {
	values := []Type{{N: 19}}
	var array Numbers
	array[2] = 23
	var pointer *Type = &Type{N: 17}
	return values[0].N + array[2] + pointer.N
}
func ReadCallback() int {
	f := Id(Unique)
	return f()
}

func ReadFields() int {
	var v struct {
		Unique  int
		Counter int
		Type    int
	}
	v.Unique, v.Counter, v.Type = 1, 2, 3
	return Id(v).Unique + v.Counter + v.Type
}
