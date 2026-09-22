package main

func checkZeroTail(n int) bool {
	for repeat := 0; repeat < 5; repeat++ {
		b := make([]byte, n)
		for i := 0; i < len(b); i++ {
			if b[i] != 0 {
				return false
			}
			b[i] = 173
		}
	}
	return true
}
func appMain(args []string) int {
	for n := 0; n < 20; n++ {
		if !checkZeroTail(n) {
			return 1
		}
	}
	print("PASS\n")
	return 0
}
