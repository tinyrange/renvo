package main

func appMain(args []string) int {
	backward := []byte{1, 2, 3, 4, 5}
	if copy(backward[1:], backward[:4]) != 4 || backward[0] != 1 || backward[1] != 1 || backward[2] != 2 || backward[3] != 3 || backward[4] != 4 {
		return 1
	}
	forward := []byte{1, 2, 3, 4, 5}
	if copy(forward[:4], forward[1:]) != 4 || forward[0] != 2 || forward[1] != 3 || forward[2] != 4 || forward[3] != 5 || forward[4] != 5 {
		return 2
	}
	print("PASS\n")
	return 0
}
