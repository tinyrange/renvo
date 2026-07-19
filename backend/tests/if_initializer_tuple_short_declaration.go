package main

func renvoIfInitializerPair() ([]byte, bool) {
	return []byte{'P'}, true
}

func appMain(args []string) int {
	if data, ok := renvoIfInitializerPair(); ok {
		if len(data) == 1 && data[0] == 'P' {
			print("PASS\n")
			return 0
		}
	}
	print("FAIL\n")
	return 1
}
