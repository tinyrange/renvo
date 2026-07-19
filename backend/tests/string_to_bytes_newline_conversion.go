package main

var renvo0578Text string = "a\nb"

func appMain(args []string) int {
	bs := []byte(renvo0578Text)
	if len(bs) != 3 || bs[1] != '\n' {
		print("RENVO-0578 newline conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
