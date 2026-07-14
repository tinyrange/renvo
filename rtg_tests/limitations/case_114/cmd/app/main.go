package main

type Inner struct { Count int; Tag string; Flags [2]bool }
type Box struct { Value int; Name string; OK bool; Ptr *int; Values [2]int; Inner Inner }
type Empty struct{}
type Values [2]int
type NamedBox struct { Values Values }
func same(left Box, right Box) bool { return left == right }
func makeBox(v int) Box { return Box{v, "a", true, nil, [2]int{1, 2}, Inner{3, "b", [2]bool{true, false}}} }
func main() { x := 1; left := Box{1, "a", true, &x, [2]int{1, 2}, Inner{3, "b", [2]bool{true, false}}}; right := Box{1, "a", true, &x, [2]int{1, 2}, Inner{3, "b", [2]bool{true, false}}}; emptyLeft := Empty{}; emptyRight := Empty{}; namedLeft := NamedBox{Values{1, 2}}; namedRight := NamedBox{Values{1, 2}}; _ = same(left, right) && left == right && !(left != right) && emptyLeft == emptyRight && namedLeft == namedRight && Box{1, "a", true, nil, [2]int{1, 2}, Inner{3, "b", [2]bool{true, false}}} == Box{1, "a", true, nil, [2]int{1, 2}, Inner{3, "b", [2]bool{true, false}}} && makeBox(1) == makeBox(1) }
