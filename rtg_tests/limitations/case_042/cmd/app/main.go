package main

type T struct{ X int }
func (t T) Inc() int { return t.X+1 }
func main() { t := T{1}; f := t.Inc; _ = f() }
