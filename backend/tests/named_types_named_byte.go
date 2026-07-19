package main

type Renvo0653Letter byte

var renvo0653Global Renvo0653Letter = 'z'

func appMain(args []string) int {
	if renvo0653Global != 'z' {
		print("RENVO-0653 named byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
