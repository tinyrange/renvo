package main

import "testing"

func TestCoreLaunchSequence(t *testing.T) {
	got := coreLaunchSequence(0x20020100)
	want := [6]uint32{0, 0, 1, 0x20010000, 0x20040000, 0x20020101}
	if got != want {
		t.Fatalf("core launch sequence = %#v, want %#v", got, want)
	}
}

func TestMonitorVersionContract(t *testing.T) {
	if protocolMajor != 2 || protocolMinor != 0 || monitorVersion != 0x00010100 {
		t.Fatalf("monitor version = protocol %d.%d firmware %#x", protocolMajor, protocolMinor, monitorVersion)
	}
	if chipRP2040 != 0x2040 || chipRP2350 != 0x2350 {
		t.Fatalf("monitor chip identifiers = %#x, %#x", chipRP2040, chipRP2350)
	}
}

func TestCoreLaunchIsBounded(t *testing.T) {
	if coreLaunchPollLimit <= 0 || coreLaunchRetryLimit <= 0 {
		t.Fatalf("core launch bounds = %d polls, %d retries", coreLaunchPollLimit, coreLaunchRetryLimit)
	}
}
