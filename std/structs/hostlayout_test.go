package structs

import (
	"testing"
	"unsafe"
)

func TestHostLayoutMatchesHostABI(t *testing.T) {
	type header struct {
		_     HostLayout
		Kind  uint8
		Value uint32
	}
	if got := unsafe.Sizeof(header{}); got != 8 {
		t.Fatalf("host-layout header size = %d, want 8", got)
	}
}
