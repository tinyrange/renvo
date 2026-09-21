package main

func good() int { return 5 }
func bad() int  { panic("unused function value candidate") }

func appMain(args []string) int {
	selected := good
	if selected() != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
