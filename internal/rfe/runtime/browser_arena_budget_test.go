//go:build renvo && browser

package runtime

import (
	"strings"
	"testing"
)

func TestBrowserEntryBudgetRecyclesBeforeHostTableExhaustion(t *testing.T) {
	n := &Native{capacity: 32 << 20}
	threshold := browserNativeEntryLimit * 7 / 8
	for count := 0; count < threshold-1; count++ {
		n.recordWasm((count+1)*16, 1, false, 1)
	}
	if n.EntryBudgetNeedsRecycle() {
		t.Fatal("entry recycling started below its threshold")
	}
	n.recordWasm(threshold*16, 1, false, 1)
	if !n.EntryBudgetNeedsRecycle() || n.Bytes >= n.capacity*7/8 {
		t.Fatal("slot pressure did not independently request recycling")
	}
	for count := threshold; count < browserNativeEntryLimit-1; count++ {
		n.recordWasm((count+1)*16, 1, false, 1)
	}
	// The last usable entry is admitted to the service. There is deliberately no
	// device here: reaching that error proves the slot check allowed the request.
	if _, err := n.exchangeWasm([]byte{0}, 1, false); err == nil || !strings.Contains(err.Error(), "closed browser arena") {
		t.Fatal("last usable slot was not admitted", err)
	}
	n.recordWasm(browserNativeEntryLimit*16, 1, false, 1)
	before := n.Bytes
	if _, err := n.exchangeWasm([]byte{0}, 1, false); err == nil || !strings.Contains(err.Error(), "entry slots full") {
		t.Fatal("exhausted table reached the host service", err)
	}
	if n.Bytes != before || n.entryCount != browserNativeEntryLimit {
		t.Fatal("rejected installation changed arena accounting")
	}
}
