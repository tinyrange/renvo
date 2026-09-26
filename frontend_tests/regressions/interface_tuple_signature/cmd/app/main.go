package main

type reader interface{ Read([]byte) (int, error) }
type readerWithName interface {
	reader
	Name() string
}
type source struct{ n int }

func (s *source) Read(p []byte) (int, error) { p[0] = 42; return s.n, nil }
func (s *source) Name() string               { return "source" }

type wrong struct{}

func (s *wrong) Read(p []byte) (error, int) { return nil, 2 }
func (s *wrong) Name() string               { return "wrong" }
func matches(v interface{}) bool            { _, ok := v.(readerWithName); return ok }
func main() {
	var v interface{} = &source{n: 1}
	if !matches(v) || matches(&wrong{}) || matches(source{}) || matches(3) {
		panic("method set")
	}
	r, ok := v.(reader)
	if !ok {
		panic("reader assertion")
	}
	p := make([]byte, 1)
	n, err := r.Read(p)
	if n != 1 || err != nil || p[0] != 42 {
		panic("tuple call")
	}
	println("PASS")
}
