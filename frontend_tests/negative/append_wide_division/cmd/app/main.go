package main

func main() {
	_ = append([]byte{}, (1<<100)/(1<<92))
}
