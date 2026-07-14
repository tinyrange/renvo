package main

type T struct { A [2]int }
func main() { t := T{A: [2]int{1,2}}; _ = len(t.A) }
