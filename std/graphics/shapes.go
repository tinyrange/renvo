package graphics

func (s *Surface) DrawPoint(point Point, color Color) {
	s.transformPointInPlace(&point)
	s.putPixel(scalarFloor(point.X), scalarFloor(point.Y), color)
}

func (s *Surface) FillPolygon(points []Point, rule FillRule, color Color) {
	if len(points) < 3 {
		return
	}
	var path Path
	path.MoveTo(points[0])
	for i := 1; i < len(points); i++ {
		path.LineTo(points[i])
	}
	path.Close()
	s.FillPath(&path, rule, color)
}

func (s *Surface) ellipseBounds(rect Rect) (Scalar, Scalar, Scalar, Scalar) {
	a := Point{X: rect.MinX, Y: rect.MinY}
	b := Point{X: rect.MaxX, Y: rect.MinY}
	c := Point{X: rect.MaxX, Y: rect.MaxY}
	d := Point{X: rect.MinX, Y: rect.MaxY}
	s.transformPointInPlace(&a)
	s.transformPointInPlace(&b)
	s.transformPointInPlace(&c)
	s.transformPointInPlace(&d)
	minX, maxX, minY, maxY := a.X, a.X, a.Y, a.Y
	points := []Point{b, c, d}
	for i := 0; i < len(points); i++ {
		p := points[i]
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return minX, maxX, minY, maxY
}

func (s *Surface) inversePoint(x, y Scalar) (Point, bool) {
	x /= s.deviceScale
	y /= s.deviceScale
	if !s.transformComplex {
		return Point{X: x - s.transformTX, Y: y - s.transformTY}, true
	}
	determinant := s.transformA*s.transformD - s.transformB*s.transformC
	if determinant == 0.0 {
		return Point{}, false
	}
	x -= s.transformTX
	y -= s.transformTY
	return Point{
		X: (s.transformD*x - s.transformC*y) / determinant,
		Y: (-s.transformB*x + s.transformA*y) / determinant,
	}, true
}

func (s *Surface) FillEllipse(rect Rect, color Color) {
	if rect.Empty() {
		return
	}
	minX, maxX, minY, maxY := s.ellipseBounds(rect)
	cx, cy := (rect.MinX+rect.MaxX)/2.0, (rect.MinY+rect.MaxY)/2.0
	rx, ry := rect.Width()/2.0, rect.Height()/2.0
	for y := scalarFloor(minY); y < scalarCeil(maxY); y++ {
		for x := scalarFloor(minX); x < scalarCeil(maxX); x++ {
			p, ok := s.inversePoint(Scalar(x)+0.5, Scalar(y)+0.5)
			if !ok {
				return
			}
			dx, dy := p.X-cx, p.Y-cy
			if dx*dx*ry*ry+dy*dy*rx*rx <= rx*rx*ry*ry {
				s.putPixel(x, y, color)
			}
		}
	}
}

func (s *Surface) StrokeEllipse(rect Rect, width Scalar, color Color) {
	if rect.Empty() || width <= 0.0 {
		return
	}
	innerRX := rect.Width()/2.0 - width
	innerRY := rect.Height()/2.0 - width
	if innerRX <= 0.0 || innerRY <= 0.0 {
		s.FillEllipse(rect, color)
		return
	}
	minX, maxX, minY, maxY := s.ellipseBounds(rect)
	cx, cy := (rect.MinX+rect.MaxX)/2.0, (rect.MinY+rect.MaxY)/2.0
	outerRX, outerRY := rect.Width()/2.0, rect.Height()/2.0
	for y := scalarFloor(minY); y < scalarCeil(maxY); y++ {
		for x := scalarFloor(minX); x < scalarCeil(maxX); x++ {
			p, ok := s.inversePoint(Scalar(x)+0.5, Scalar(y)+0.5)
			if !ok {
				return
			}
			dx, dy := p.X-cx, p.Y-cy
			outer := dx*dx*outerRY*outerRY+dy*dy*outerRX*outerRX <= outerRX*outerRX*outerRY*outerRY
			inner := dx*dx*innerRY*innerRY+dy*dy*innerRX*innerRX < innerRX*innerRX*innerRY*innerRY
			if outer && !inner {
				s.putPixel(x, y, color)
			}
		}
	}
}
