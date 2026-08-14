// Package st7121 decodes and filters touch reports from an ST7121 controller.
package st7121

const (
	// Address is the controller's I2C address.
	Address = byte(0x55)
	// NativeWidth is the controller's portrait raster width.
	NativeWidth = 720
	// NativeHeight is the controller's portrait raster height.
	NativeHeight = 1280
	// MaximumContacts is the largest report supported by the controller.
	MaximumContacts = 10
)

// Point is one touch contact in the native portrait orientation.
type Point struct {
	ID        int
	X         int
	Y         int
	Strength  int
	Intensity int
}

// ReportStats describes the most recently consumed controller report.
type ReportStats struct {
	SensingCounter int
	Advanced       int
	RawCount       int
	Checksum       int
	CoordSum       int
	ReportSum      int
	CoordXOR       int
	Reports        int
}

// ST7121 firmware 1.80.1.16 occasionally reports coherent phantom contacts
// while the integrated display scanner is active. They are not damaged I2C
// packets: the same reports are returned by the vendor ESP-IDF driver. Keep
// established tracks responsive, but debounce new secondary contacts so the
// controller's short edge pairs and low-area grids never reach applications.

const touchTrackDistanceSquared = 128 * 128
const touchPendingDistanceSquared = 96 * 96
const touchSecondaryConfirm = 2
const touchWeakConfirm = 3
const touchReplacementConfirm = 2
const touchTrackMissingLimit = 3
const touchSecondaryIntensityMinimum = 28
const touchReplacementIntensityMinimum = 20
const touchLowIntensityTrackDistanceSquared = 48 * 48

type filteredTouchTrack struct {
	point   Point
	active  bool
	primary bool
	missing int
}

type pendingTouchTrack struct {
	point  Point
	streak int
	seen   bool
}

// Filter suppresses scanner-coupled phantom contacts while preserving tracked
// fingers and rapid primary-contact replacement.
type Filter struct {
	tracks      [10]filteredTouchTrack
	pending     [10]pendingTouchTrack
	replacement pendingTouchTrack
}

func touchDistanceSquared(left Point, right Point) int {
	dx := left.X - right.X
	dy := left.Y - right.Y
	return dx*dx + dy*dy
}

func touchMirrorPair(left Point, right Point) bool {
	if left.X > right.X {
		left, right = right, left
	}
	return left.X < 96 && right.X >= NativeWidth-96 &&
		left.X+right.X >= NativeWidth-112 &&
		left.X+right.X <= NativeWidth+80
}

func touchKnownLoneGhost(point Point) bool {
	// Captured single-frame artifacts cluster at the two native X rails around
	// the vertical midpoint and normally have area one or two. Preserve a short
	// confirmation only for that signature; applying it to every distant lone
	// contact is what made rapid taps at the landscape ends feel unresponsive.
	return point.Strength <= 2 &&
		(point.X < 8 || point.X >= NativeWidth-8) &&
		point.Y >= NativeHeight/2-96 && point.Y <= NativeHeight/2+96
}

func touchSecondaryIntensityTrusted(point Point) bool {
	// Firmware 1.80.1.16's coupled secondary images can have plausible area and
	// coordinates, but the captured intensity remains below the real contact.
	// Zero keeps synthetic/unit-test points and older reports without intensity
	// compatible; this ST7121 always supplies a nonzero value on active records.
	return point.Intensity == 0 || point.Intensity >= touchSecondaryIntensityMinimum
}

func touchTrackCandidateTrusted(track Point, candidate Point, distance int) bool {
	// Do not let a nearby low-intensity secondary image take over a strong
	// established finger during a one-frame dropout. Genuine release pressure
	// can decay below the threshold, but then its displacement remains small.
	return candidate.Intensity == 0 || candidate.Intensity >= touchSecondaryIntensityMinimum ||
		track.Intensity < touchSecondaryIntensityMinimum ||
		distance <= touchLowIntensityTrackDistanceSquared
}

// Reset forgets all established and pending contacts.
func (filter *Filter) Reset() {
	for index := 0; index < len(filter.tracks); index++ {
		filter.tracks[index].active = false
		filter.tracks[index].primary = false
		filter.tracks[index].missing = 0
		filter.pending[index].streak = 0
		filter.pending[index].seen = false
	}
	filter.replacement.streak = 0
	filter.replacement.seen = false
}

func (filter *Filter) activeCount() int {
	count := 0
	for index := 0; index < len(filter.tracks); index++ {
		if filter.tracks[index].active {
			count++
		}
	}
	return count
}

func (filter *Filter) allocate(point Point, primary bool) int {
	for index := 0; index < len(filter.tracks); index++ {
		if !filter.tracks[index].active {
			point.ID = index
			filter.tracks[index].point = point
			filter.tracks[index].active = true
			filter.tracks[index].primary = primary
			filter.tracks[index].missing = 0
			return index
		}
	}
	return -1
}

func appendFilteredTouch(points []Point, count int, point Point) int {
	if count < len(points) {
		points[count] = point
		return count + 1
	}
	return count
}

// Apply filters one raw controller frame into caller-provided storage.
func (filter *Filter) Apply(raw []Point, points []Point) int {
	if len(raw) == 0 {
		filter.Reset()
		return 0
	}

	var used [10]bool
	count := 0
	primaryMatched := false
	// Existing tracks win over every noise heuristic. This keeps a real finger
	// responsive even when it moves through the edge band where ghosts appear.
	for trackID := 0; trackID < len(filter.tracks); trackID++ {
		track := &filter.tracks[trackID]
		if !track.active {
			continue
		}
		best := -1
		bestDistance := touchTrackDistanceSquared + 1
		for rawIndex := 0; rawIndex < len(raw); rawIndex++ {
			if used[rawIndex] || raw[rawIndex].Strength <= 0 || raw[rawIndex].Strength > 24 {
				continue
			}
			// A confirmed secondary remains conditional on the evidence which
			// admitted it. Do not keep replaying a scanner image after its intensity
			// drops or it disappears; only the primary gets dropout tolerance.
			if !track.primary && !touchSecondaryIntensityTrusted(raw[rawIndex]) {
				continue
			}
			distance := touchDistanceSquared(track.point, raw[rawIndex])
			if !touchTrackCandidateTrusted(track.point, raw[rawIndex], distance) {
				continue
			}
			if distance < bestDistance {
				best = rawIndex
				bestDistance = distance
			}
		}
		if best < 0 || bestDistance > touchTrackDistanceSquared {
			track.missing++
			missingLimit := touchTrackMissingLimit
			if !track.primary {
				missingLimit = 0
			}
			if track.missing > missingLimit {
				track.active = false
				track.primary = false
				continue
			}
			// Preserve the last trustworthy coordinate across the ST7121's brief
			// bad frames. Re-emitting it does not draw because it is unchanged.
			count = appendFilteredTouch(points, count, track.point)
			continue
		}
		used[best] = true
		if track.primary {
			primaryMatched = true
		}
		track.missing = 0
		track.point = raw[best]
		track.point.ID = trackID
		count = appendFilteredTouch(points, count, track.point)
	}

	// A lone first contact is genuine in captured failure traces: the ST7121
	// only starts synthesizing its mirrored contacts after that touch exists.
	if filter.activeCount() == 0 && len(raw) == 1 && !used[0] &&
		raw[0].Strength > 0 && raw[0].Strength <= 24 {
		trackID := filter.allocate(raw[0], true)
		if trackID >= 0 {
			used[0] = true
			count = appendFilteredTouch(points, count, filter.tracks[trackID].point)
		}
	}

	// The protocol has no contact-down/contact-up state. A rapid lift and
	// retouch between two roughly 9 ms sensing frames therefore looks exactly
	// like one contact jumping a long distance, often from one physical end of
	// the landscape screen to the other. If the old track did not match and the
	// controller reports one compact point, replace the stale primary instead of
	// treating the new tap as a secondary touch. Only the known weak midpoint
	// rail artifact still requires a second frame.
	if !primaryMatched && filter.activeCount() > 0 && len(raw) == 1 && !used[0] &&
		raw[0].Strength > 0 && raw[0].Strength <= 16 &&
		(raw[0].Intensity == 0 || raw[0].Intensity >= touchReplacementIntensityMinimum) {
		distance := touchDistanceSquared(filter.replacement.point, raw[0])
		if filter.replacement.streak == 0 || distance > touchPendingDistanceSquared {
			filter.replacement.streak = 1
		} else {
			filter.replacement.streak++
		}
		filter.replacement.point = raw[0]
		filter.replacement.seen = true
		required := 1
		if touchKnownLoneGhost(raw[0]) {
			required = touchReplacementConfirm
		}
		if filter.replacement.streak >= required {
			point := raw[0]
			filter.Reset()
			trackID := filter.allocate(point, true)
			if trackID >= 0 {
				count = 0
				used[0] = true
				count = appendFilteredTouch(points, count, filter.tracks[trackID].point)
			}
		}
	} else {
		filter.replacement.streak = 0
		filter.replacement.seen = false
	}

	var mirrored [10]bool
	for left := 0; left < len(raw); left++ {
		if used[left] {
			continue
		}
		for right := left + 1; right < len(raw); right++ {
			if !used[right] && touchMirrorPair(raw[left], raw[right]) {
				mirrored[left] = true
				mirrored[right] = true
			}
		}
	}

	for index := 0; index < len(filter.pending); index++ {
		filter.pending[index].seen = false
	}
	for rawIndex := 0; rawIndex < len(raw); rawIndex++ {
		if used[rawIndex] || mirrored[rawIndex] {
			continue
		}
		point := raw[rawIndex]
		// Captured real fingers are compact (normally area 3..15). Large new
		// contacts are the scanner-coupled 17/32/34/48/64 edge artifacts. Smaller
		// coupled images are rejected by their consistently lower intensity.
		if point.Strength <= 0 || point.Strength > 24 ||
			!touchSecondaryIntensityTrusted(point) {
			continue
		}
		best := -1
		bestDistance := touchPendingDistanceSquared + 1
		for pendingID := 0; pendingID < len(filter.pending); pendingID++ {
			pending := &filter.pending[pendingID]
			if pending.streak == 0 || pending.seen {
				continue
			}
			distance := touchDistanceSquared(pending.point, point)
			if distance < bestDistance {
				best = pendingID
				bestDistance = distance
			}
		}
		if best < 0 || bestDistance > touchPendingDistanceSquared {
			for pendingID := 0; pendingID < len(filter.pending); pendingID++ {
				if filter.pending[pendingID].streak == 0 {
					best = pendingID
					break
				}
			}
			if best < 0 {
				continue
			}
			filter.pending[best].streak = 0
		}
		pending := &filter.pending[best]
		pending.point = point
		pending.streak++
		pending.seen = true
		required := touchSecondaryConfirm
		if point.Strength <= 2 {
			required = touchWeakConfirm
		}
		if pending.streak >= required {
			trackID := filter.allocate(point, false)
			if trackID >= 0 {
				count = appendFilteredTouch(points, count, filter.tracks[trackID].point)
			}
			pending.streak = 0
		}
	}
	for index := 0; index < len(filter.pending); index++ {
		if !filter.pending[index].seen {
			filter.pending[index].streak = 0
		}
	}
	return count
}
