//go:build !renvo

package graphics

func glyphRow(r, y int) byte { return glyphRows(r)[y] }
