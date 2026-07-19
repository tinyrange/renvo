package main

func appMain(args []string) int {
	count := 0
	for i := 0; i < 3; i = i + 1 {
		switch i {
		case 1:
			break
		default:
			count = count + 10
		}
		count = count + 1
	}
	if count != 23 {
		print("RENVO-SWITCH-015 break scope failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
