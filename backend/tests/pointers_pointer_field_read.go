package main

type Renvo0635Box struct{ value int }

func appMain(args []string) int {
	box := Renvo0635Box{value: 19}
	p := &box
	for {
		if p.value == 19 {
			break
		}
	}
	if p.value != 19 {
		print("RENVO-0635 pointer field read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
