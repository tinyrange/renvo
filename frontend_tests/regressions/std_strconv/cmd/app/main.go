package main

import "strconv"

func main() {
	if strconv.FormatInt(-9223372036854775807-1, 10) != "-9223372036854775808" {
		print("FAIL\n")
		return
	}
	value, err := strconv.ParseInt("-128", 10, 8)
	if err != nil || value != -128 {
		print("FAIL\n")
		return
	}
	value, err = strconv.ParseInt("128", 10, 8)
	if value != 127 || err == nil {
		print("FAIL\n")
		return
	}
	u, err := strconv.ParseUint("0b1010_0101", 0, 8)
	if err != nil || u != 165 {
		print("FAIL\n")
		return
	}
	u, err = strconv.ParseUint("256", 10, 8)
	if u != 255 || err == nil {
		print("FAIL\n")
		return
	}
	floating, err := strconv.ParseFloat("-2.5e2", 64)
	if err != nil || floating != -250 || strconv.FormatFloat(12.5, 'f', 2, 64) != "12.50" || strconv.FormatFloat(12.5, 'g', -1, 64) != "12.5" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
