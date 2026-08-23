//go:build !renvo

package strconv

import stdstrconv "strconv"

func ParseFloat(s string, bitSize int) (float64, error) {
	return stdstrconv.ParseFloat(s, bitSize)
}

func FormatFloat(f float64, fmt byte, prec int, bitSize int) string {
	return stdstrconv.FormatFloat(f, fmt, prec, bitSize)
}

func AppendFloat(dst []byte, f float64, fmt byte, prec int, bitSize int) []byte {
	return stdstrconv.AppendFloat(dst, f, fmt, prec, bitSize)
}
