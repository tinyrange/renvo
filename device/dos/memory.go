package dos

// Segment identifies a real-mode memory segment. It is intentionally distinct
// from a Go pointer, which remains a near offset in the program's data segment.
type Segment uint16

func (s Segment) Load8(offset uint16) byte { return segmentLoad8(uint16(s), offset) }
func (s Segment) Store8(offset uint16, value byte) {
	segmentStore8(uint16(s), offset, value)
}
func (s Segment) Fill8(offset uint16, value byte, count uint16) {
	segmentFill8(uint16(s), offset, value, count)
}
func (s Segment) Write(offset uint16, data []byte) {
	segmentWrite(uint16(s), offset, data)
}

func AllocateSegment(paragraphs uint16) (Segment, error) {
	segment, err := AllocateParagraphs(paragraphs)
	return Segment(segment), err
}

func (s Segment) Free() error { return FreeParagraphs(uint16(s)) }
func (s Segment) Resize(paragraphs uint16) error {
	return ResizeParagraphs(uint16(s), paragraphs)
}
