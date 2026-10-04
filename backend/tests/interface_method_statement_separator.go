package main

// Keep both methods on one source line to exercise the explicit separator.
type SeparatorReader interface {
	First() int; Second() int
}

type SeparatorValue int

func (v SeparatorValue) First() int  { return int(v) }
func (v SeparatorValue) Second() int { return int(v) + 1 }

func separatorRead(v SeparatorReader) int { return v.Second() }

func appMain() int {
	var v SeparatorReader = SeparatorValue(41)
	if separatorRead(v) != 42 || v.First() != 41 {
		return 1
	}
	second := v.Second
	if second() != 42 {
		return 2
	}
	print("PASS\n")
	return 0
}
