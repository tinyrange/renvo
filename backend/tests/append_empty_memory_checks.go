package main

func intBounds() (caught bool) {
	defer func() {
		if recover() != nil {
			caught = true
		}
	}()
	var rows [][]int
	var empty []int
	rows[0] = append(rows[0], empty...)
	return false
}

func byteBounds() (caught bool) {
	defer func() {
		if recover() != nil {
			caught = true
		}
	}()
	var rows [][]byte
	var empty []byte
	rows[0] = append(rows[0], empty...)
	return false
}

func appMain() int {
	if !intBounds() || !byteBounds() {
		return 1
	}
	print("PASS\n")
	return 0
}
