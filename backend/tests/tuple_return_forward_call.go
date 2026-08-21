package main

func splitOnce(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return len(data), data, nil
}

func splitTwice(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return len(data), data, nil
}

func forwardSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF {
		return splitOnce(data, atEOF)
	}
	if len(data) != 0 {
		return splitTwice(data, atEOF)
	}
	var zeroAdvance int
	var zeroToken []byte
	var zeroErr error
	return zeroAdvance, zeroToken, zeroErr
}

func appMain() int {
	advance, token, err := forwardSplit([]byte("word"), true)
	if advance == 4 && string(token) == "word" && err == nil {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
