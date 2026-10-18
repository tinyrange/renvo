package main

func equalWord(a int, b int) bool { return a == b }
func appMain() int {
	if equalWord(1048574, 1048573) {
		return 1
	}
	if !equalWord(1048574, 1048574) {
		return 2
	}
	if equalWord(4096, 4095) || equalWord(65536, 65535) {
		return 3
	}
	print("PASS\n")
	return 0
}
