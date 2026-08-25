package main

import (
	"renvo.dev/device/board"
	"renvo.dev/std/graphics"
	"renvo.dev/std/strconv"
)

var colors = []graphics.Color{
	graphics.RGBA(0x36, 0xc5, 0xf0, 0xff),
	graphics.RGBA(0xff, 0x5c, 0x7a, 0xff),
	graphics.RGBA(0x54, 0xd9, 0x83, 0xff),
	graphics.RGBA(0xff, 0xc1, 0x3d, 0xff),
	graphics.RGBA(0xb0, 0x76, 0xff, 0xff),
	graphics.RGBA(0xff, 0x8b, 0x3d, 0xff),
	graphics.RGBA(0x44, 0xe0, 0xd0, 0xff),
	graphics.RGBA(0xf4, 0x70, 0xc5, 0xff),
	graphics.RGBA(0xa5, 0xdb, 0x55, 0xff),
	graphics.RGBA(0x69, 0x8f, 0xff, 0xff),
}

func reportSuffix(stats board.TouchReportStats) string {
	return " SCAN " + strconv.Itoa(stats.SensingCounter) +
		" ADV " + strconv.Itoa(stats.Advanced) +
		" CK " + strconv.Itoa(stats.Checksum) +
		" SUM " + strconv.Itoa(stats.CoordSum) +
		" RSUM " + strconv.Itoa(stats.ReportSum) +
		" XOR " + strconv.Itoa(stats.CoordXOR)
}

func logPoints(points []board.TouchPoint, count int, stats board.TouchReportStats) {
	print("TOUCHES " + strconv.Itoa(count) + reportSuffix(stats) + "\n")
	for index := 0; index < count; index++ {
		point := points[index]
		print("TOUCH " + strconv.Itoa(point.ID) + " " +
			strconv.Itoa(point.X) + "," + strconv.Itoa(point.Y) +
			" AREA " + strconv.Itoa(point.Strength) +
			" INTENSITY " + strconv.Itoa(point.Intensity) + "\n")
	}
}

func sameTouchReports(raw []board.TouchPoint, rawCount int, filtered []board.TouchPoint, filteredCount int) bool {
	if rawCount != filteredCount {
		return false
	}
	for rawIndex := 0; rawIndex < rawCount; rawIndex++ {
		found := false
		for filteredIndex := 0; filteredIndex < filteredCount; filteredIndex++ {
			if raw[rawIndex].X == filtered[filteredIndex].X &&
				raw[rawIndex].Y == filtered[filteredIndex].Y &&
				raw[rawIndex].Strength == filtered[filteredIndex].Strength {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func logFilteredRaw(points []board.TouchPoint, count int, stats board.TouchReportStats) {
	print("RAW FILTERED " + strconv.Itoa(count) + reportSuffix(stats) + "\n")
	for index := 0; index < count; index++ {
		point := points[index]
		print("RAW " + strconv.Itoa(point.ID) + " " +
			strconv.Itoa(point.X) + "," + strconv.Itoa(point.Y) +
			" AREA " + strconv.Itoa(point.Strength) +
			" INTENSITY " + strconv.Itoa(point.Intensity) + "\n")
	}
}

func pointChanged(previous [10]board.TouchPoint, active [10]bool, point board.TouchPoint) bool {
	return !active[point.ID] || previous[point.ID].X != point.X ||
		previous[point.ID].Y != point.Y || previous[point.ID].Strength != point.Strength
}

func main() {
	print("TAB5 TOUCH TRAILS MAIN\n")
	if !board.InitFramebuffer() || !board.InitTouch() {
		print("TAB5 TOUCH TRAILS INIT FAIL\n")
		for {
		}
	}
	firmware, miscellaneous := board.TouchProtocolInfo()
	print("TAB5 TOUCH FW " + strconv.Itoa(firmware) +
		" MISC " + strconv.Itoa(miscellaneous) + "\n")
	if miscellaneous&16 != 0 {
		print("TAB5 TOUCH COORD CHECKSUM SUPPORTED\n")
	} else {
		print("TAB5 TOUCH COORD CHECKSUM UNAVAILABLE\n")
	}
	surface := board.NewPortraitSurface()
	if surface == nil {
		print("TAB5 TOUCH TRAILS SURFACE FAIL\n")
		for {
		}
	}
	canvas := surface
	canvas.Clear(graphics.RGBA(0x05, 0x08, 0x10, 0xff))
	if !board.PresentPortrait(surface) {
		print("TAB5 TOUCH TRAILS PRESENT FAIL\n")
		for {
		}
	}
	surface.ResetDirty()
	print("TAB5 TOUCH TRAILS PASS\n")

	var previous [10]board.TouchPoint
	var active [10]bool
	for {
		var points [10]board.TouchPoint
		count, ok := board.ReadTouches(points[:])
		if !ok {
			print("TAB5 TOUCH TRAILS READ FAIL\n")
			continue
		}
		var raw [10]board.TouchPoint
		rawCount := board.TouchRawReport(raw[:])
		stats := board.TouchLastReportStats()
		if !sameTouchReports(raw[:], rawCount, points[:], count) {
			logFilteredRaw(raw[:], rawCount, stats)
		}
		changed := false
		var nextActive [10]bool
		for index := 0; index < count; index++ {
			point := points[index]
			if point.ID < 0 || point.ID >= len(nextActive) {
				continue
			}
			nextActive[point.ID] = true
			if pointChanged(previous, active, point) {
				if active[point.ID] {
					canvas.DrawLine(
						graphics.Point{X: graphics.Scalar(previous[point.ID].X), Y: graphics.Scalar(previous[point.ID].Y)},
						graphics.Point{X: graphics.Scalar(point.X), Y: graphics.Scalar(point.Y)},
						9, colors[point.ID],
					)
				}
				canvas.FillEllipse(graphics.R(
					graphics.Scalar(point.X-7), graphics.Scalar(point.Y-7), 14, 14,
				), colors[point.ID])
				changed = true
			}
			previous[point.ID] = point
		}
		for id := 0; id < len(active); id++ {
			if active[id] != nextActive[id] {
				changed = true
			}
			active[id] = nextActive[id]
		}
		if changed {
			logPoints(points[:], count, stats)
			if board.PresentPortrait(surface) {
				surface.ResetDirty()
			}
		}
		board.Refresh()
	}
}
