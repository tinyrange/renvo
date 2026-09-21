package main

type DistantFields struct {
	padding [320]byte
	b       byte
	s       int16
	u       uint16
	i       int32
	w       uint32
	x       int64
}

func fillFields(p *DistantFields) {
	p.b = 251
	p.s = -1234
	p.u = 65000
	p.i = -1234567
	p.w = 4000000000
	p.x = -1234567890123
}

func appMain(args []string) int {
	p := new(DistantFields)
	fillFields(p)
	if p.b != 251 {
		return 11
	}
	if p.s != -1234 {
		println(p.s)
		return 12
	}
	if p.u != 65000 {
		return 13
	}
	if p.i != -1234567 {
		return 14
	}
	if p.w != 4000000000 {
		return 15
	}
	if p.x != -1234567890123 {
		return 16
	}
	if p.padding[319] != 0 {
		return 2
	}
	print("PASS\n")
	return 0
}
