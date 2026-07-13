package main

func main() {
	s := make([]int, 1, 1)
	s[0] = 3
	t := append(s, 4)
	if s[0] == 3 && t[0] == 3 && t[1] == 4 && len(t) == 2 && cap(t) >= 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
