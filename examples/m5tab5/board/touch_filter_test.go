package board

import "testing"

func applyTouchFrame(filter *touchFilter, raw ...TouchPoint) []TouchPoint {
	points := make([]TouchPoint, 10)
	count := filter.apply(raw, points)
	return points[:count]
}

func TestTouchFilterKeepsPrimaryAndRejectsMirroredGhosts(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 401, Y: 912, Strength: 6}
	points := applyTouchFrame(&filter, primary)
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("initial points = %#v, want primary", points)
	}

	for frame := 0; frame < 20; frame++ {
		primary.Y--
		points = applyTouchFrame(&filter, primary,
			TouchPoint{X: 0, Y: 641, Strength: 32},
			TouchPoint{X: 719, Y: 637, Strength: 32})
		if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
			t.Fatalf("frame %d points = %#v, want moving primary", frame, points)
		}
	}
}

func TestTouchFilterConfirmsGenuineSecondContact(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 360, Y: 900, Strength: 7}
	secondary := TouchPoint{X: 500, Y: 300, Strength: 8}
	applyTouchFrame(&filter, primary)
	for frame := 1; frame < touchSecondaryConfirm; frame++ {
		points := applyTouchFrame(&filter, primary, secondary)
		if len(points) != 1 {
			t.Fatalf("frame %d points = %#v, want only established contact", frame, points)
		}
	}
	points := applyTouchFrame(&filter, primary, secondary)
	if len(points) != 2 {
		t.Fatalf("confirmed points = %#v, want two contacts", points)
	}
	if points[0].ID == points[1].ID {
		t.Fatalf("confirmed contacts share ID %d", points[0].ID)
	}
}

func TestTouchFilterConfirmsGenuineSecondContactAtPhysicalEdge(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 360, Y: 900, Strength: 7}
	secondary := TouchPoint{X: 2, Y: 300, Strength: 8}
	applyTouchFrame(&filter, primary)
	for frame := 1; frame < touchSecondaryConfirm; frame++ {
		points := applyTouchFrame(&filter, primary, secondary)
		if len(points) != 1 {
			t.Fatalf("frame %d points = %#v, want only established contact", frame, points)
		}
	}
	points := applyTouchFrame(&filter, primary, secondary)
	if len(points) != 2 {
		t.Fatalf("edge points = %#v, want two contacts", points)
	}
}

func TestTouchFilterAllowsRealPrimaryAtEdge(t *testing.T) {
	var filter touchFilter
	points := applyTouchFrame(&filter, TouchPoint{X: 2, Y: 640, Strength: 8})
	if len(points) != 1 || points[0].X != 2 {
		t.Fatalf("edge primary = %#v, want immediate real contact", points)
	}
}

func TestTouchFilterReleaseClearsTracks(t *testing.T) {
	var filter touchFilter
	applyTouchFrame(&filter, TouchPoint{X: 300, Y: 400, Strength: 8})
	if points := applyTouchFrame(&filter); len(points) != 0 {
		t.Fatalf("release points = %#v, want none", points)
	}
	points := applyTouchFrame(&filter, TouchPoint{X: 600, Y: 700, Strength: 7})
	if len(points) != 1 || points[0].ID != 0 {
		t.Fatalf("new primary = %#v, want reset track zero", points)
	}
}

func TestTouchFilterDoesNotLetBadFrameHijackPrimary(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 146, Y: 1070, Strength: 4}
	applyTouchFrame(&filter, primary)

	badFrames := []TouchPoint{
		{X: 145, Y: 1060, Strength: 36},
		{X: 139, Y: 1006, Strength: 36},
		{X: 108, Y: 845, Strength: 35},
	}
	for frame, bad := range badFrames {
		points := applyTouchFrame(&filter, bad)
		if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
			t.Fatalf("bad frame %d points = %#v, want held primary", frame, points)
		}
	}

	primary = TouchPoint{X: 130, Y: 1045, Strength: 6}
	points := applyTouchFrame(&filter, primary)
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("recovered points = %#v, want primary", points)
	}
}

func TestTouchFilterRapidRetouchReplacesStalePrimaryImmediately(t *testing.T) {
	var filter touchFilter
	oldPoint := TouchPoint{X: 360, Y: 1200, Strength: 7}
	newPoint := TouchPoint{X: 360, Y: 20, Strength: 6}
	applyTouchFrame(&filter, oldPoint)

	points := applyTouchFrame(&filter, newPoint)
	if len(points) != 1 || points[0].Y != newPoint.Y || points[0].ID != 0 {
		t.Fatalf("replacement = %#v, want immediate new primary", points)
	}
}

func TestTouchFilterDoesNotReplacePrimaryWithSingleWeakGlitch(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 360, Y: 900, Strength: 7}
	applyTouchFrame(&filter, primary)
	points := applyTouchFrame(&filter, TouchPoint{X: 719, Y: 640, Strength: 2})
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("glitch points = %#v, want held primary", points)
	}
	points = applyTouchFrame(&filter, primary)
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("recovered points = %#v, want primary", points)
	}
}

func TestTouchFilterConfirmsPersistentWeakRailContact(t *testing.T) {
	var filter touchFilter
	oldPoint := TouchPoint{X: 360, Y: 900, Strength: 7}
	weakPoint := TouchPoint{X: 719, Y: 640, Strength: 2}
	applyTouchFrame(&filter, oldPoint)

	points := applyTouchFrame(&filter, weakPoint)
	if len(points) != 1 || points[0].X != oldPoint.X || points[0].Y != oldPoint.Y {
		t.Fatalf("first weak frame = %#v, want held primary", points)
	}
	points = applyTouchFrame(&filter, weakPoint)
	if len(points) != 1 || points[0].X != weakPoint.X || points[0].Y != weakPoint.Y {
		t.Fatalf("persistent weak contact = %#v, want replacement", points)
	}
}

func TestTouchFilterRejectsLowIntensitySecondaryImage(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 500, Y: 800, Strength: 10, Intensity: 48}
	ghost := TouchPoint{X: 590, Y: 820, Strength: 6, Intensity: 18}
	applyTouchFrame(&filter, primary)
	for frame := 0; frame < 8; frame++ {
		points := applyTouchFrame(&filter, primary, ghost)
		if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
			t.Fatalf("frame %d points = %#v, want only primary", frame, points)
		}
	}
}

func TestTouchFilterAllowsHighIntensitySecondaryContact(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 500, Y: 800, Strength: 10, Intensity: 48}
	secondary := TouchPoint{X: 590, Y: 820, Strength: 6, Intensity: 31}
	applyTouchFrame(&filter, primary)
	for frame := 1; frame < touchSecondaryConfirm; frame++ {
		points := applyTouchFrame(&filter, primary, secondary)
		if len(points) != 1 {
			t.Fatalf("frame %d points = %#v, want only primary", frame, points)
		}
	}
	points := applyTouchFrame(&filter, primary, secondary)
	if len(points) != 2 {
		t.Fatalf("confirmed points = %#v, want two contacts", points)
	}
}

func TestTouchFilterDropsSecondaryAsSoonAsItsIntensityBecomesUntrusted(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 500, Y: 800, Strength: 10, Intensity: 48}
	secondary := TouchPoint{X: 590, Y: 820, Strength: 6, Intensity: 31}
	applyTouchFrame(&filter, primary)
	applyTouchFrame(&filter, primary, secondary)
	points := applyTouchFrame(&filter, primary, secondary)
	if len(points) != 2 {
		t.Fatalf("confirmed points = %#v, want two contacts", points)
	}

	secondary.Intensity = 19
	points = applyTouchFrame(&filter, primary, secondary)
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("low-intensity frame = %#v, want only primary", points)
	}
}

func TestTouchFilterDoesNotReplayMissingSecondary(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 500, Y: 800, Strength: 10, Intensity: 48}
	secondary := TouchPoint{X: 590, Y: 820, Strength: 6, Intensity: 31}
	applyTouchFrame(&filter, primary)
	applyTouchFrame(&filter, primary, secondary)
	points := applyTouchFrame(&filter, primary, secondary)
	if len(points) != 2 {
		t.Fatalf("confirmed points = %#v, want two contacts", points)
	}

	points = applyTouchFrame(&filter, primary)
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("missing-secondary frame = %#v, want only primary", points)
	}
}

func TestTouchFilterDoesNotLetLowIntensityImageHijackTrack(t *testing.T) {
	var filter touchFilter
	primary := TouchPoint{X: 675, Y: 792, Strength: 8, Intensity: 49}
	ghost := TouchPoint{X: 594, Y: 811, Strength: 6, Intensity: 15}
	applyTouchFrame(&filter, primary)

	points := applyTouchFrame(&filter, ghost)
	if len(points) != 1 || points[0].X != primary.X || points[0].Y != primary.Y {
		t.Fatalf("points = %#v, want held primary", points)
	}
}
