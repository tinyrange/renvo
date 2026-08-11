//go:build renvo && linux && amd64

package main

func syscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return 0
}
