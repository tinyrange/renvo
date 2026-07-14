package main

type T struct{ X int }
func (t T) Inc() int { return t.X+1 }
func main() { f := T.Inc; _ = T.Inc; _ = f; _ = f(T{1}) }
