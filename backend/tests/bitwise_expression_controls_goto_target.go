package main

func appMain(args []string) int {
	x := 0x08
	if x&0x08 != 0 {
		goto good
	}
	print("RENVO-0225 bitwise_expression_controls_goto_target failed\n")
	return 1
good:
	print("PASS\n")
	return 0
}
