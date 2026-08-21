//go:build !renvo

package time

import "syscall"

func Now() Time {
	var tv syscall.Timeval
	if err := syscall.Gettimeofday(&tv); err != nil {
		return Time{}
	}
	return Unix(int64(tv.Sec), int64(tv.Usec)*1000)
}
