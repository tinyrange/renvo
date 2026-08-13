package mtp

import "testing"

func TestContainerRoundTrip(t *testing.T) {
	payload := []byte{1, 2, 3, 4}
	wire := make([]byte, 12+len(payload))
	written, err := Encode(Container{Type: ContainerCommand, Code: 0x1001, Transaction: 9}, payload, wire)
	if err != nil || written != len(wire) {
		t.Fatalf("Encode = %d, %v", written, err)
	}
	var container Container
	decoded := make([]byte, len(payload))
	count, err := Decode(wire, &container, decoded)
	if err != nil || count != len(payload) || container.Type != ContainerCommand || container.Code != 0x1001 || container.Transaction != 9 || string(decoded) != string(payload) {
		t.Fatalf("Decode = %#v, %v, %d, %v", container, decoded, count, err)
	}
}
