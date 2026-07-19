package main

func appMain() int {
	first := open("renvo_fd_first.tmp", O_RDWR|O_CREATE|O_TRUNC)
	second := open("renvo_fd_second.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if first < 0 || second < 0 || first == second {
		return 1
	}
	if close(first) != 0 {
		return 2
	}
	third := open("renvo_fd_third.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if third < 0 || third == second {
		return 3
	}
	if write(second, []byte("SECOND"), -1) != 6 {
		return 4
	}
	if write(third, []byte("THIRD"), -1) != 5 {
		return 5
	}
	if close(second) != 0 || close(third) != 0 {
		return 6
	}

	second = open("renvo_fd_second.tmp", O_RDONLY)
	third = open("renvo_fd_third.tmp", O_RDONLY)
	if second < 0 || third < 0 || second == third {
		return 7
	}
	secondData := make([]byte, 6)
	thirdData := make([]byte, 5)
	if read(second, secondData, -1) != 6 || string(secondData) != "SECOND" {
		return 8
	}
	if read(third, thirdData, -1) != 5 || string(thirdData) != "THIRD" {
		return 9
	}
	if close(second) != 0 || close(third) != 0 {
		return 10
	}
	print("PASS\n")
	return 0
}
