package model

type Hidden struct{ value int }
type Blocker int

func (v Blocker) value() int { return int(v) }
func NewHidden() Hidden      { return Hidden{7} }
