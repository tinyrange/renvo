package main

func appMain(args []string) int {
	s := "ab"
	i := 0
	state := 0
start:
	if state == 0 {
		if s[i] != byte(97) {
			goto fail
		}
		i = i + 1
		state = 1
		goto start
	}
	if state == 1 {
		if s[i] != byte(98) {
			goto fail
		}
		goto pass
	}
fail:
	print("RENVO-0475 parser failed\n")
	return 1
pass:
	print("PASS\n")
	return 0
}
