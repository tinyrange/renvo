package main

type Payload struct {
	Tag  byte
	Data [4096]byte
	Tail uint64
}

func makePayload(tag byte, tail uint64) *Payload { return &Payload{Tag: tag, Tail: tail} }
func appMain(args []string) int {
	a := makePayload(7, 0x8877665544332211)
	b := makePayload(9, 123)
	for i := 0; i < len(a.Data); i++ {
		if a.Data[i] != 0 || b.Data[i] != 0 {
			print("FAIL composite zero\n")
			return 1
		}
	}
	a.Data[2048] = 42
	if a.Tag != 7 || a.Tail != 0x8877665544332211 || b.Tag != 9 || b.Tail != 123 || b.Data[2048] != 0 {
		print("FAIL composite fields\n")
		return 1
	}
	print("PASS\n")
	return 0
}
