package main

type shape20Alias = int

func shape20Add(v shape20Alias) int { return v + 12 }
func appMain(args []string) int {
	var x shape20Alias = 8
	if shape20Add(x) != 20 {
		print("program_shape_20 alias\n")
		return 1
	}
	print("PASS\n")
	return 0
}
