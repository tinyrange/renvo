package main

import "testing"

func TestControlConversion(t *testing.T) {
	if controlByte('c') != 0x03 || controlByte('[') != 0x1b || controlByte('1') != '1' {
		t.Fatal("control conversion mismatch")
	}
}
