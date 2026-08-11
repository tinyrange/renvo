package midi

import "testing"

func TestEventPacketRoundTrip(t *testing.T) {
	wire := make([]byte, 4)
	want := Event{Cable: 2, Code: 9, Byte0: 0x90, Byte1: 60, Byte2: 127}
	if !want.Bytes(wire) {
		t.Fatal("Bytes rejected valid event")
	}
	var got Event
	if !Parse(wire, &got) || got != want {
		t.Fatalf("Parse = %#v from %v", got, wire)
	}
}
