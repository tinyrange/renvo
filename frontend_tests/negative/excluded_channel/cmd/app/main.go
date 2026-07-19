package main

var values chan int

func main() { _ = <-values }
