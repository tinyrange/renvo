package main

type groupedNamed int

type groupedHead struct {
	Head int
}

type groupedTail struct {
	Tail int
}

type groupedPoint struct {
	groupedHead
	X, Y        int
	Label       string
	Left, Right groupedNamed `json:"coordinates"`
	*groupedTail
}

type groupedReturnValue struct {
	X, Y int
}

func (point *groupedPoint) move(dx, dy int) {
	point.X += dx
	point.Y += dy
}

func groupedReturn(point groupedReturnValue) groupedReturnValue {
	return point
}

func appMain(args []string) int {
	tail := groupedTail{Tail: 9}
	point := groupedPoint{groupedHead{Head: 1}, 2, 3, "point", 4, 5, &tail}
	point.move(10, 20)
	if point.Head != 1 || point.X != 12 || point.Y != 23 || point.Label != "point" {
		return 1
	}
	if point.Left != groupedNamed(4) {
		return 1
	}
	if point.Right != groupedNamed(5) {
		return 1
	}
	if point.groupedTail.Tail != 9 {
		return 1
	}
	moved := groupedReturn(groupedReturnValue{X: point.X, Y: point.Y})
	if moved.X != 12 || moved.Y != 23 {
		return 1
	}
	keyed := groupedPoint{X: 6, Y: 7, Left: groupedNamed(8), Right: groupedNamed(9)}
	if keyed.X != 6 || keyed.Y != 7 || keyed.Left != groupedNamed(8) || keyed.Right != groupedNamed(9) {
		return 1
	}
	print("PASS\n")
	return 0
}
