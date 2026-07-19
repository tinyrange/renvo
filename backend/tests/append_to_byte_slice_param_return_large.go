package main

func fillParam(out []byte, n int, tag byte) []byte {
	for i := 0; i < n; i++ {
		out = append(out, byte(i%251))
	}
	out[0] = tag
	out[n-1] = tag
	return out
}

func appMain(args []string, env []string) int {
	out := make([]byte, 0, 90000)
	got := fillParam(out, 90000, 'Z')
	if len(got) != 90000 {
		print("FAIL len\n")
		return 1
	}
	if got[0] != 'Z' || got[89999] != 'Z' || got[12345] != byte(12345%251) {
		print("FAIL data\n")
		return 1
	}
	print("PASS\n")
	return 0
}
