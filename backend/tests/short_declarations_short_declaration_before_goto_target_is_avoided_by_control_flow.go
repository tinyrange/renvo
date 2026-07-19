package main

func appMain(args []string) int {
	x := 1
	goto target
	{
		y := 99
		x = y
	}
target:
	if !(x == 1) {
		print("RENVO-0325 short_declaration_before_goto_target_is_avoided_by_control_flow failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
