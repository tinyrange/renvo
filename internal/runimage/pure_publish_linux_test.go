//go:build !renvo && linux && (amd64 || arm64)

package runimage

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func mappingPermissions(t *testing.T, address uintptr) string {
	t.Helper()
	maps, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(maps), "\n") {
		var start, end uintptr
		var permissions string
		if _, err := fmt.Sscanf(line, "%x-%x %s", &start, &end, &permissions); err == nil && start <= address && address < end {
			return permissions
		}
	}
	t.Fatalf("no mapping for native address %x", address)
	return ""
}

func TestAppendPublicationOnlySealsTouchedPages(t *testing.T) {
	page := os.Getpagesize()
	arena, err := NewCodeArena(page * 16)
	if err != nil {
		t.Fatal(err)
	}
	defer arena.Close()
	check := func(address uintptr, writable, executable bool) {
		t.Helper()
		permissions := mappingPermissions(t, address)
		if strings.Contains(permissions, "w") != writable || strings.Contains(permissions, "x") != executable {
			t.Fatalf("native permissions %s, want writable=%v executable=%v", permissions, writable, executable)
		}
	}
	// Publication must not flip the entire arena RX and then back RW for each
	// tiny append. Unpublished pages stay non-executable, published pages RX.
	if _, err = arena.InstallBlock([]byte{0, 0, 0, 0}, 1); err != nil {
		t.Fatal(err)
	}
	check(arena.base, false, true)
	check(arena.base+uintptr(page*8), true, false)
	// Append across a host page boundary, including a partly published page.
	if _, err = arena.InstallBlock(make([]byte, page*2), 1); err != nil {
		t.Fatal(err)
	}
	check(arena.base, false, true)
	check(arena.base+uintptr(page*2), false, true)
	check(arena.base+uintptr(page*8), true, false)
	if arena.used > page*3 || arena.size != page*16 {
		t.Fatal("publication expanded arena bounds")
	}
}
