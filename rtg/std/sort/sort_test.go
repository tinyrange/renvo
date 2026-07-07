package sort

import "testing"

func TestSortAndSearch(t *testing.T) {
	ss := []string{"b", "a", "c"}
	Strings(ss)
	if ss[0] != "a" || ss[2] != "c" {
		t.Fatalf("Strings failed: %#v", ss)
	}
	is := []int{4, 1, 3, 2}
	Ints(is)
	if is[0] != 1 || is[3] != 4 {
		t.Fatalf("Ints failed: %#v", is)
	}
	if Search(10, func(i int) bool { return i >= 6 }) != 6 {
		t.Fatalf("Search failed")
	}
}
