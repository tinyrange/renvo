package model

type C interface{ ~int }
type Reader interface{ Value() int }
type Box int

func (b Box) Value() int { return int(b) }
func Read(v Reader) int  { return v.Value() }
func Shadow(v int) int {
	C := func(v int) int { return v + 1 }
	return C(v)
}
