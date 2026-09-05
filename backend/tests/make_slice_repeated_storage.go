package main

type storage struct { values []int }
func initStorage(s *storage) { s.values = make([]int, 128) }

func appMain(args []string) int {
	var slices [][]int
	for i := 0; i < 6; i++ {
		var box storage
		initStorage(&box)
		s := box.values
		if s[0] != 0 { panic("make zero") }
		s[0] = i+1
		slices = append(slices, s)
	}
	for i := 0; i < len(slices); i++ {
		if slices[i][0] != i+1 { panic("make reused live storage") }
	}
	print("PASS\n")
	return 0
}
