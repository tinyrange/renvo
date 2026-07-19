package main

func byteLit09(b byte) bool { return b == 'D' }
func appMain(args []string) int {
	if !byteLit09('D') {
		print("byte_literals_09 param\n")
		return 1
	}
	print("PASS\n")
	return 0
}
