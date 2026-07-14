package main

type Inner struct { X int }
type Outer struct { Inner }
func main() { o := Outer{Inner{1}}; _ = o.X; _ = Outer{Inner{2}}.X }
