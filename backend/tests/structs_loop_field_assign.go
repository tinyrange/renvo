package main

type Renvo0615Counter struct{ value int }

func appMain(args []string) int {
	c := Renvo0615Counter{}
	p := &c
	for i := 0; i < 4; i = i + 1 {
		p.value = p.value + i
	}
	if c.value != 6 {
		print("RENVO-0615 loop field assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
