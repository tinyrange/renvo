package main

func dynamicPreserveLength() int { return 32 }

func appMain(args []string) int {
	first := make([]byte, dynamicPreserveLength())
	for i := 0; i < len(first); i++ {
		first[i] = byte(i + 1)
	}
	second := make([]byte, dynamicPreserveLength())
	for i := 0; i < len(second); i++ {
		if second[i] != 0 || first[i] != byte(i+1) {
			print("FAIL\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
