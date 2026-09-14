package button

import "testing"

func TestDebouncedPressReleaseAndWrap(t *testing.T) {
	b := Debouncer{DelayMS: 20}
	for _, sample := range []struct {
		now                     uint32
		pressed, changed, state bool
	}{
		{0xfffffff0, true, false, false},
		{0xfffffff8, false, false, false},
		{0xfffffffc, true, false, false},
		{15, true, false, false},
		{16, true, true, true},
		{100, true, false, true},
		{101, false, false, true},
		{105, true, false, true},
		{110, false, false, true},
		{130, false, true, false},
	} {
		if changed := b.Update(sample.pressed, sample.now); changed != sample.changed || b.Pressed != sample.state {
			t.Fatalf("sample %+v: changed=%v pressed=%v", sample, changed, b.Pressed)
		}
	}
}
