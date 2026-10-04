package main

import "example.com/genericslices/slices"
import "example.com/genericslices/maps"

type Integers []int
type Record struct{ Key, Sequence int }

func sorting() {
	numbers := make(Integers, 257)
	for i := range numbers {
		numbers[i] = (i * 37) % len(numbers)
	}
	slices.Sort(numbers)
	for i, v := range numbers {
		if v != i {
			panic("ordered sort permutation")
		}
	}
	if !slices.IsSorted(numbers) || slices.Min(numbers) != 0 || slices.Max(numbers) != 256 {
		panic("ordered extrema")
	}
	index, found := slices.BinarySearch(numbers, 42)
	if index != 42 || !found {
		panic("ordered search")
	}
	index, found = slices.BinarySearch(numbers, 300)
	if index != len(numbers) || found {
		panic("ordered missing search")
	}
	words := []string{"z", "a", "c", "b"}
	slices.Sort(words)
	if !slices.Equal(words, []string{"a", "b", "c", "z"}) {
		panic("string sort")
	}
	direction := -1
	slices.SortFunc(numbers, func(a, b int) int { return direction * (a - b) })
	for i, v := range numbers {
		if v != 256-i {
			panic("callback descending sort")
		}
	}
	records := make([]Record, 128)
	for i := range records {
		records[i] = Record{(i * 13) % 7, i}
	}
	compare := func(a, b Record) int { return a.Key - b.Key }
	slices.SortStableFunc(records, compare)
	for i := 1; i < len(records); i++ {
		a, b := records[i-1], records[i]
		if a.Key > b.Key || a.Key == b.Key && a.Sequence > b.Sequence {
			panic("stable record sort")
		}
	}
	if !slices.IsSortedFunc(records, compare) || slices.MinFunc(records, compare).Key != 0 || slices.MaxFunc(records, compare).Key != 6 {
		panic("callback extrema")
	}
	index, found = slices.BinarySearchFunc(records, 3, func(a Record, key int) int { return a.Key - key })
	if !found || records[index].Key != 3 || index > 0 && records[index-1].Key >= 3 {
		panic("callback search")
	}
	zero := 0.0
	nan := zero / zero
	floats := []float64{3, nan, -1, 0}
	slices.Sort(floats)
	if floats[0] == floats[0] || floats[1] != -1 || floats[2] != 0 || floats[3] != 3 {
		panic("float sort")
	}
	index, found = slices.BinarySearch(floats, nan)
	if index != 0 || !found {
		panic("NaN search")
	}
}

func main() {
	sorting()
	iterators()
	var absent Integers
	if slices.Clone(absent) != nil || slices.Concat(absent, absent) != nil {
		panic("nil preservation")
	}
	s := slices.Clone(Integers{1, 2, 3})
	slices.Reverse(s)
	if !slices.Equal(s, Integers{3, 2, 1}) || !slices.Contains(s, 2) || slices.Index(s, 1) != 2 {
		panic("clone reverse search")
	}
	s = slices.Insert(s, 1, 8, 9)
	s = slices.Delete(s, 2, 4)
	if !slices.Equal(s, Integers{3, 8, 1}) {
		panic("insert delete")
	}
	s = slices.Replace(s, 1, 2, 5, 6)
	if !slices.Equal(s, Integers{3, 5, 6, 1}) {
		panic("replace")
	}
	storage := make([]int, 6, 16)
	copy(storage, []int{1, 2, 3, 4, 5, 6})
	storage = slices.Insert(storage, 2, storage[1:4]...)
	if !slices.Equal(storage, []int{1, 2, 2, 3, 4, 3, 4, 5, 6}) {
		panic("overlapping insert")
	}
	storage = slices.Replace(storage, 1, 5, storage[5:]...)
	if !slices.Equal(storage, []int{1, 3, 4, 5, 6, 3, 4, 5, 6}) {
		panic("overlapping replace")
	}
	limit := 4
	if slices.IndexFunc(s, func(v int) bool { return v > limit }) != 1 || !slices.ContainsFunc(s, func(v int) bool { return v == 6 }) {
		panic("callback search")
	}
	s = slices.DeleteFunc(s, func(v int) bool { return v > limit })
	if !slices.Equal(s, Integers{3, 1}) {
		panic("callback delete")
	}
	if !slices.Equal(slices.Compact([]int{1, 1, 2, 2, 3}), []int{1, 2, 3}) {
		panic("compact")
	}
	words := slices.CompactFunc([]string{"a", "b", "cc", "dd", "eee"}, func(a, b string) bool { return len(a) == len(b) })
	if !slices.Equal(words, []string{"a", "cc", "eee"}) || !slices.EqualFunc([]int{1, 2, 3}, words, func(a int, b string) bool { return a == len(b) }) {
		panic("callback equality")
	}
	if slices.Compare([]string{"a", "c"}, []string{"a", "b"}) <= 0 || slices.CompareFunc([]int{1, 2}, []string{"a", "bb"}, func(a int, b string) int { return a - len(b) }) != 0 {
		panic("comparison")
	}
	grown := slices.Grow(s, 20)
	if len(grown) != 2 || cap(grown) < 22 || cap(slices.Clip(grown)) != 2 {
		panic("capacity")
	}
	joined := slices.Concat(Integers{1, 2}, Integers{3}, nil)
	if !slices.Equal(joined, Integers{1, 2, 3}) || !slices.Equal(slices.Repeat(joined, 2), Integers{1, 2, 3, 1, 2, 3}) {
		panic("concat repeat")
	}
	print("PASS\n")
}

func iterators() {
	items := Integers{3, 1, 2}
	sum := 0
	for i, v := range slices.All(items) {
		sum += i + v
	}
	if sum != 9 {
		panic("slice All")
	}
	if !slices.Equal(slices.Collect(slices.Values(items)), []int{3, 1, 2}) {
		panic("slice Collect")
	}
	back := []int{}
	for _, v := range slices.Backward(items) {
		back = append(back, v)
	}
	if !slices.Equal(back, []int{2, 1, 3}) {
		panic("slice Backward")
	}
	if !slices.Equal(slices.AppendSeq(Integers{4}, slices.Values(items)), Integers{4, 3, 1, 2}) {
		panic("slice AppendSeq")
	}
	if !slices.Equal(slices.Sorted(slices.Values(items)), []int{1, 2, 3}) {
		panic("slice Sorted")
	}
	if !slices.Equal(slices.SortedFunc(slices.Values(items), func(a, b int) int { return b - a }), []int{3, 2, 1}) {
		panic("slice SortedFunc")
	}
	records := []Record{{2, 0}, {1, 1}, {2, 2}}
	sorted := slices.SortedStableFunc(slices.Values(records), func(a, b Record) int { return a.Key - b.Key })
	if len(sorted) != 3 || sorted[0].Sequence != 1 || sorted[1].Sequence != 0 || sorted[2].Sequence != 2 {
		panic("slice SortedStableFunc")
	}
	chunks := slices.Collect(slices.Chunk(items, 2))
	if len(chunks) != 2 || !slices.Equal(chunks[0], Integers{3, 1}) || !slices.Equal(chunks[1], Integers{2}) || cap(chunks[0]) != 2 {
		panic("slice Chunk")
	}
	var absent Integers
	if slices.Collect(slices.Values(absent)) != nil {
		panic("empty iterator nil preservation")
	}
	original := map[string]int{"a": 1, "b": 2}
	collected := maps.Collect(maps.All(original))
	if len(collected) != 2 || collected["a"] != 1 || collected["b"] != 2 {
		panic("map Collect and All")
	}
	maps.Insert(collected, func(yield func(string, int) bool) { yield("a", 7); yield("c", 3) })
	if len(collected) != 3 || collected["a"] != 7 || collected["c"] != 3 {
		panic("map Insert")
	}
	keys := slices.Sorted(maps.Keys(collected))
	if !slices.Equal(keys, []string{"a", "b", "c"}) {
		panic("map Keys")
	}
	if !slices.Equal(slices.Sorted(maps.Values(collected)), []int{2, 3, 7}) {
		panic("map Values")
	}
}
