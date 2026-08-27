package main

import "testing"

func TestCoreLaunchSequence(t *testing.T) {
	got := coreLaunchSequence(0x20020100)
	want := [6]uint32{0, 0, 1, 0x20010000, 0x20040000, 0x20020101}
	if got != want {
		t.Fatalf("core launch sequence = %#v, want %#v", got, want)
	}
}
