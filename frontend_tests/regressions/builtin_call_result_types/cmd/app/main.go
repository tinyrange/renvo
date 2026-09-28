package main

type Reader struct{}

func (r *Reader) read() rune        { return '雪' }
func pair() (count int, value rune) { return 3, '☃' }
func main() {
	reader := &Reader{}
	r := reader.read()
	var data []byte
	data = append(data, string(r)...)
	n, other := pair()
	data = append(data, string(other)...)
	if string(data) != "雪☃" || n != 3 || string(n) != "\x03" {
		panic("call result types")
	}
	print("PASS\n")
}
