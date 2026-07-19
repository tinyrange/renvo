package main

type namedString string

func appMain(args []string) int {
	value := namedString("PASS\n")
	text := string(value)
	if text != "PASS\n" {
		print(text)
		return 1
	}
	print(text)
	return 0
}
