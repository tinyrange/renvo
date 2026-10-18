package main

type dynamicSliceSource interface{ View() []byte }
type dynamicSliceBox struct{ data []byte }

func (b *dynamicSliceBox) View() []byte                           { return b.data }
func dynamicSliceInterface(s dynamicSliceSource) []byte           { return s.View() }
func dynamicSliceFunction(f func() []byte) []byte                 { return f() }
func dynamicSliceLocal(s dynamicSliceSource) []byte               { value := s.View(); return value[1:3:4] }
func dynamicSliceIdentity(p []byte) []byte                        { return p }
func dynamicSliceArgument(f func([]byte) []byte, p []byte) []byte { return f(p) }
func dynamicSliceEscaped(value byte) []byte {
	var data [4]byte
	data[0] = value
	return dynamicSliceArgument(dynamicSliceIdentity, data[:])
}
func appMain() int {
	data := make([]byte, 3, 6)
	box := &dynamicSliceBox{data: data}
	a := dynamicSliceInterface(box)
	b := dynamicSliceFunction(box.View)
	c := dynamicSliceLocal(box)
	d := dynamicSliceArgument(dynamicSliceIdentity, data)
	data[0] = 17
	if a[0] != 17 || b[0] != 17 || d[0] != 17 || cap(a) != 6 || cap(b) != 6 || cap(d) != 6 || len(c) != 2 || cap(c) != 3 {
		print("FAIL alias/capacity\n")
		return 1
	}
	a[1] = 23
	if data[1] != 23 || b[1] != 23 || c[0] != 23 || d[1] != 23 {
		print("FAIL bidirectional alias\n")
		return 1
	}
	a = append(a, 31)
	if data[:4][3] != 31 {
		print("FAIL spare capacity alias\n")
		return 1
	}
	escaped := dynamicSliceEscaped(41)
	dynamicSliceEscaped(51)
	dynamicSliceEscaped(61)
	if escaped[0] != 41 {
		print("FAIL local array escape\n")
		return 1
	}
	print("PASS\n")
	return 0
}
