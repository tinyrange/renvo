package main

import "fmt"

func main() {
	values := fmt.Sprint("values:", int8(-2), uint8(3), int16(-4), uint16(5), int32(-6), uint32(7))
	formatted := fmt.Sprintf("formatted: %d %d %x %s %t %%", int16(-8), uint16(9), uint8(0xe5), "ok", true)
	if values != "values:-2 3 -4 5 -6 7" || formatted != "formatted: -8 9 e5 ok true %" {
		print("FAIL\n")
		return
	}
	fmt.Printf("%s\n", "PASS")
}
