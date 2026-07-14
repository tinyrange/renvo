package main

func main() { outer: for i := 0; i < 3; i++ { for j := 0; j < 3; j++ { if j == 1 { continue outer }; break outer } } }
