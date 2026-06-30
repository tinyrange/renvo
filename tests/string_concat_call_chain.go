package main

type errString string

func word(v int) string {
	if v == 7 {
		return "7"
	}
	return "x"
}

func appMain(args []string) int {
	out := errString("a" + ":" + word(7) + ":" + word(8) + ": " + "z")
	if out != "a:7:x: z" {
		print(string(out))
		print("\n")
		return 1
	}
	print("PASS\n")
	return 0
}
