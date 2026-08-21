package main

import "time"

func main() {
	start := time.Now()
	for i := 0; i < 100000; i++ {
	}
	elapsed := time.Since(start)
	if start.Year() < 2020 || elapsed < 0 || elapsed > time.Minute {
		print("FAIL\n")
		return
	}
	t, err := time.Parse(time.RFC3339Nano, "2024-02-29T12:34:56.123456789+05:30")
	if err != nil || t.Format(time.RFC3339Nano) != "2024-02-29T12:34:56.123456789+05:30" {
		print("FAIL\n")
		return
	}
	u := t.UTC().Add(2*time.Second + 750*time.Nanosecond)
	if u.Unix() != 1709190298 || u.Nanosecond() != 123457539 || !u.After(t) || u.Sub(t) != 2*time.Second+750*time.Nanosecond {
		print("FAIL\n")
		return
	}
	y, m, d := u.Date()
	if y != 2024 || m != time.February || d != 29 || u.Hour() != 7 || u.Minute() != 4 || u.Second() != 58 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
