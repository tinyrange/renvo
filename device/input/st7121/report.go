package st7121

const (
	reportOffset     = 0
	coordinateOffset = 4
	coordinateSize   = 7
)

// ReportSize returns the byte count needed for one atomic touch report.
func ReportSize(maximum int) int {
	if maximum < 0 || maximum > MaximumContacts {
		return 0
	}
	return coordinateOffset + maximum*coordinateSize
}

// DecodeReport decodes one report-page snapshot into caller-provided storage.
// Coordinates outside the supplied display geometry are ignored.
func DecodeReport(page []byte, maximum, width, height int, points []Point) (int, int, bool) {
	size := ReportSize(maximum)
	if size == 0 || len(page) < size {
		return 0, 0, false
	}
	advanced := int(page[reportOffset])
	if page[reportOffset]&0x08 == 0 {
		return advanced, 0, true
	}
	count := 0
	for contact := 0; contact < maximum; contact++ {
		offset := coordinateOffset + contact*coordinateSize
		if page[offset]&0x80 == 0 {
			continue
		}
		point := Point{
			ID:        contact,
			X:         int(page[offset]&0x3f)<<8 | int(page[offset+1]),
			Y:         int(page[offset+2])<<8 | int(page[offset+3]),
			Strength:  int(page[offset+4]),
			Intensity: int(page[offset+5]),
		}
		if point.X < 0 || point.X >= width || point.Y < 0 || point.Y >= height {
			continue
		}
		if count < len(points) {
			points[count] = point
			count++
		}
	}
	return advanced, count, true
}
