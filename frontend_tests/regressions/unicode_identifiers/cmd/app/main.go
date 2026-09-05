package main

import (
	α "example.com/renvotests/regressions/unicode_identifiers/a"
	β "example.com/renvotests/regressions/unicode_identifiers/b"
)

func main() {
	__renvo_unicode_cf80 := 99
	π := α.Ω()
	世界 := β.Ω()
	α٢ := π + 世界
	𐐀 := α.Σ{Φ: α٢}
	if 𐐀.Φ != 7 || __renvo_unicode_cf80 != 99 {
		panic("Unicode identifier resolution")
	}
	println("PASS")
}
