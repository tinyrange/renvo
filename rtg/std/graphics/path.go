package graphics

type FillRule int

const (
	FillEvenOdd FillRule = iota
	FillNonZero
)

// Path stores backend-neutral line contours. Curves are flattened when they
// are appended so every platform backend sees the same geometry.
type Path struct {
	points []Point
	starts []int
	closed []bool
}

func (p *Path) Reset() {
	p.points = nil
	p.starts = nil
	p.closed = nil
}

func (p *Path) MoveTo(point Point) {
	p.starts = append(p.starts, len(p.points))
	p.closed = append(p.closed, false)
	p.points = append(p.points, point)
}

func (p *Path) LineTo(point Point) {
	if len(p.starts) == 0 {
		p.MoveTo(point)
		return
	}
	p.points = append(p.points, point)
}

func (p *Path) current() (Point, bool) {
	if len(p.points) == 0 {
		return Point{}, false
	}
	return p.points[len(p.points)-1], true
}

func midpoint(a, b Point) Point {
	return Point{X: (a.X + b.X) / 2.0, Y: (a.Y + b.Y) / 2.0}
}

func flattenQuad(path *Path, start, control, end Point, depth int) {
	if depth == 0 {
		path.LineTo(end)
		return
	}
	left := midpoint(start, control)
	right := midpoint(control, end)
	middle := midpoint(left, right)
	flattenQuad(path, start, left, middle, depth-1)
	flattenQuad(path, middle, right, end, depth-1)
}

func (p *Path) QuadTo(control, end Point) {
	start, ok := p.current()
	if !ok {
		p.MoveTo(end)
		return
	}
	flattenQuad(p, start, control, end, 4)
}

func flattenCubic(path *Path, start, control1, control2, end Point, depth int) {
	if depth == 0 {
		path.LineTo(end)
		return
	}
	a := midpoint(start, control1)
	b := midpoint(control1, control2)
	c := midpoint(control2, end)
	d := midpoint(a, b)
	e := midpoint(b, c)
	middle := midpoint(d, e)
	flattenCubic(path, start, a, d, middle, depth-1)
	flattenCubic(path, middle, e, c, end, depth-1)
}

func (p *Path) CubicTo(control1, control2, end Point) {
	start, ok := p.current()
	if !ok {
		p.MoveTo(end)
		return
	}
	flattenCubic(p, start, control1, control2, end, 4)
}

func (p *Path) Close() {
	if len(p.closed) != 0 {
		p.closed[len(p.closed)-1] = true
	}
}

func (p *Path) contourEnd(contour int) int {
	if contour+1 < len(p.starts) {
		return p.starts[contour+1]
	}
	return len(p.points)
}

func (s *Surface) StrokePath(path *Path, width Scalar, color Color) {
	if path == nil {
		return
	}
	for contour := 0; contour < len(path.starts); contour++ {
		start := path.starts[contour]
		end := path.contourEnd(contour)
		for i := start + 1; i < end; i++ {
			s.DrawLine(path.points[i-1], path.points[i], width, color)
		}
		if end-start > 1 && path.closed[contour] {
			s.DrawLine(path.points[end-1], path.points[start], width, color)
		}
	}
}

func (p *Path) Contains(point Point, rule FillRule) bool {
	if rule == FillNonZero {
		return p.pathCount(point, true) != 0
	}
	return p.pathCount(point, false)%2 != 0
}

func (p *Path) pathCount(point Point, useWinding bool) int {
	return p.pathCountTransformed(point, useWinding, nil)
}

func (p *Path) pathCountTransformed(point Point, useWinding bool, transform *Surface) int {
	if p == nil {
		return 0
	}
	count := 0
	for contour := 0; contour < len(p.starts); contour++ {
		start := p.starts[contour]
		end := p.contourEnd(contour)
		if end-start < 2 {
			continue
		}
		for i := start; i < end; i++ {
			previousIndex := i - 1
			if i == start {
				previousIndex = end - 1
			}
			previous := p.points[previousIndex]
			current := p.points[i]
			if transform != nil {
				transform.transformPointInPlace(&previous)
				transform.transformPointInPlace(&current)
			}
			direction := 0
			if (previous.Y <= point.Y && current.Y > point.Y) || (current.Y <= point.Y && previous.Y > point.Y) {
				x := previous.X + (point.Y-previous.Y)*(current.X-previous.X)/(current.Y-previous.Y)
				if x > point.X {
					direction = -1
					if current.Y > previous.Y {
						direction = 1
					}
				}
			}
			if direction != 0 {
				if useWinding {
					count += direction
				} else {
					count++
				}
			}
		}
	}
	return count
}

// FillPath closes each contour and rasterises it using the requested fill
// rule. This is the portable fallback for general, concave polygons.
func (s *Surface) FillPath(path *Path, rule FillRule, color Color) {
	if path == nil || len(path.points) < 3 {
		return
	}
	first := path.points[0]
	var transform *Surface
	if !s.transformIsIdentity() {
		transform = s
		transform.transformPointInPlace(&first)
	}
	minX, maxX, minY, maxY := first.X, first.X, first.Y, first.Y
	for i := 1; i < len(path.points); i++ {
		point := path.points[i]
		if transform != nil {
			transform.transformPointInPlace(&point)
		}
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	for y := scalarFloor(minY); y < scalarCeil(maxY); y++ {
		for x := scalarFloor(minX); x < scalarCeil(maxX); x++ {
			point := Point{X: Scalar(x) + 0.5, Y: Scalar(y) + 0.5}
			count := path.pathCountTransformed(point, rule == FillNonZero, transform)
			inside := count%2 != 0
			if rule == FillNonZero {
				inside = count != 0
			}
			if inside {
				s.putPixel(x, y, color)
			}
		}
	}
}
