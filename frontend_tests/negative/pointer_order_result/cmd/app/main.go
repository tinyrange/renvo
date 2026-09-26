package main

func pointer() *int { return nil }
func main() {
	_ = pointer() <= pointer()
}
