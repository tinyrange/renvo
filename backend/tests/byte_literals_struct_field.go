package main

type byteLit21Box struct{ mark byte }

func appMain(args []string) int {
	x := byteLit21Box{mark: 'R'}
	if x.mark != 82 {
		print("byte_literals_21 struct\n")
		return 1
	}
	print("PASS\n")
	return 0
}
