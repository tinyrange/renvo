package main

type Text string

func indexed(input string, i int) string {
	value := input[i]
	return "%" + string(value)
}
func named(input Text) string {
	value := input[0]
	return string(value)
}
func sliced(input string) string {
	value := input[1:3]
	return string(value)
}
func runeAt(input []rune) string {
	value := input[0]
	return string(value)
}
func main() {
	if indexed("xd", 1) != "%d" || named("z") != "z" || sliced("abcd") != "bc" || runeAt([]rune{0x1f600}) != "😀" {
		panic("indexed local conversion")
	}
	println("PASS")
}
