package adb

import "testing"

func TestMessageRoundTrip(t *testing.T) {
	payload := []byte("shell,v2")
	wire := make([]byte, 24+len(payload))
	written, err := Encode(Message{Command: CommandOpen, Argument0: 7, Argument1: 9}, payload, wire)
	if err != nil || written != len(wire) {
		t.Fatalf("Encode = %d, %v", written, err)
	}
	var message Message
	decoded := make([]byte, len(payload))
	count, err := Decode(wire, &message, decoded)
	if err != nil || count != len(payload) || message.Command != CommandOpen || message.Argument0 != 7 || message.Argument1 != 9 || string(decoded) != string(payload) {
		t.Fatalf("Decode = %#v, %q, %d, %v", message, decoded, count, err)
	}
}

func TestMessageRejectsBadMagicAndChecksum(t *testing.T) {
	payload := []byte("x")
	wire := make([]byte, 25)
	_, _ = Encode(Message{Command: CommandWrite}, payload, wire)
	wire[20] ^= 1
	if _, err := Decode(wire, &Message{}, make([]byte, 1)); err != ErrMalformed {
		t.Fatalf("bad magic error = %v", err)
	}
	_, _ = Encode(Message{Command: CommandWrite}, payload, wire)
	wire[24] ^= 1
	if _, err := Decode(wire, &Message{}, make([]byte, 1)); err != ErrMalformed {
		t.Fatalf("bad checksum error = %v", err)
	}
}
