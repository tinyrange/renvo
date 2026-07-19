package main

func sumMany(a int, b int, c int, d int, e int, f int, g int, h int, i int) int {
	return a*100000000 + b*10000000 + c*1000000 + d*100000 + e*10000 + f*1000 + g*100 + h*10 + i
}

func sliceAndMany(xs []byte, a int, b int, c int, d int, e int) int {
	return len(xs)*100000 + int(xs[0])*10000 + a*1000 + b*100 + c*10 + d + e
}

func appMain() int {
	xs := []byte("Z")
	a := sumMany(1, 2, 3, 4, 5, 6, 7, 8, 9)
	b := sliceAndMany(xs, 1, 2, 3, 4, 5)
	if a == 123456789 && b == 100000+900000+1000+200+30+4+5 {
		print("PASS\n")
	}
	return 0
}
