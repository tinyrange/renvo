package main

func rtgSSIPTargetCode(target string) int {
	if target == "linux/amd64" {
		return 1
	}
	if target == "linux/aarch64" {
		return 2
	}
	return 0
}

func appMain(args []string) int {
	values := make([]string, 2)
	values[0] = "-t"
	values[1] = "linux/aarch64"

	i := 1
	if rtgSSIPTargetCode(values[i]) != 2 {
		print("string_slice_index_passed_to_helper direct call failed\n")
		return 1
	}

	target := values[i]
	if rtgSSIPTargetCode(target) != 2 {
		print("string_slice_index_passed_to_helper local call failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
