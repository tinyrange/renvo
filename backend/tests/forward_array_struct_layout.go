package main

type Container struct {
	Slots [3]Slot
	Guard [256]byte
}

type Slot struct {
	Payload [37]byte
	Value   int
}

func appMain(args []string) int {
	var c Container
	for i := 0; i < 256; i++ {
		c.Guard[i] = 91
	}
	for i := 0; i < len(c.Slots); i++ {
		s := &c.Slots[i]
		s.Payload[36] = byte(i + 10)
		s.Value = i + 20
	}
	for i := 0; i < len(c.Slots); i++ {
		s := &c.Slots[i]
		if s.Payload[36] != byte(i+10) || s.Value != i+20 {
			print("FAIL slots\n")
			return 1
		}
	}
	for i := 0; i < 256; i++ {
		if c.Guard[i] != 91 {
			print("FAIL guard\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
