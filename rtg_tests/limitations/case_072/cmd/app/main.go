package main

func main() { type T int; type Box struct{ X T }; var x Box = Box{X: 1}; _ = x }
