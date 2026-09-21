package main

type NarrowField struct {
	padding byte
	value   int16
}

func setNarrowField(p *NarrowField) { p.value = -1234 }

func appMain(args []string) int {
	p := new(NarrowField)
	setNarrowField(p)
	if p.value != -1234 {
		return 1
	}
	if p.value >= -1000 {
		return 2
	}
	print("PASS\n")
	return 0
}
