package main

type Small int8
type Alias = Small

const Top = 255

func main() {
	a := append([]Alias{}, -128, 127)
	if a[0] != -128 || a[1] != 127 {
		panic("signed")
	}
	b := append([]byte{}, 0, Top, 256-1)
	if b[0] != 0 || b[1] != 255 || b[2] != 255 {
		panic("byte")
	}
	c := append([]int64{}, -9223372036854775808, 9223372036854775807)
	if c[0] >= 0 || c[1] <= 0 || c[0]+c[1] != -1 {
		panic("int64")
	}
	d := append([]uint64{}, 18446744073709551615)
	if d[0] != 18446744073709551615 {
		panic("uint64")
	}
	{
		Top := 1
		e := append([]int{}, Top)
		if e[0] != 1 {
			panic("scope")
		}
	}
	println("PASS")
}
