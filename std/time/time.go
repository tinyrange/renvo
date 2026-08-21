package time

import "errors"

type Month int

const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

type Location struct {
	name   string
	offset int
}

var utcLocation = Location{name: "UTC"}
var UTC = &utcLocation

type Time struct {
	sec  int64
	nsec int32
	loc  *Location
	mono int64
}

const (
	RFC3339     = "2006-01-02T15:04:05Z07:00"
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
)

func Unix(sec int64, nsec int64) Time {
	sec += nsec / 1000000000
	nsec %= 1000000000
	if nsec < 0 {
		nsec += 1000000000
		sec--
	}
	return Time{sec: sec, nsec: int32(nsec), loc: UTC}
}

func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time {
	if loc == nil {
		panic("time: missing Location in call to Date")
	}
	m := int(month) - 1
	year += m / 12
	m %= 12
	if m < 0 {
		m += 12
		year--
	}
	days := daysFromCivil(year, m+1, 1) + int64(day-1)
	seconds := days*86400 + int64(hour)*3600 + int64(min)*60 + int64(sec) - int64(loc.offset)
	t := Unix(seconds, int64(nsec))
	t.loc = loc
	return t
}

func (t Time) Unix() int64     { return t.sec }
func (t Time) Nanosecond() int { return int(t.nsec) }
func (t Time) Add(d Duration) Time {
	result := Unix(t.sec+int64(d)/1000000000, int64(t.nsec)+int64(d)%1000000000).in(t.loc)
	if t.mono != 0 {
		result.mono = t.mono + int64(d)
	}
	return result
}
func Since(t Time) Duration { return Now().Sub(t) }
func (t Time) Sub(u Time) Duration {
	if t.mono != 0 && u.mono != 0 {
		return Duration(t.mono - u.mono)
	}
	return Duration((t.sec-u.sec)*1000000000 + int64(t.nsec-u.nsec))
}
func (t Time) Before(u Time) bool    { return t.sec < u.sec || t.sec == u.sec && t.nsec < u.nsec }
func (t Time) After(u Time) bool     { return u.Before(t) }
func (t Time) Equal(u Time) bool     { return t.sec == u.sec && t.nsec == u.nsec }
func (t Time) UTC() Time             { return t.in(UTC) }
func (t Time) in(loc *Location) Time { t.loc = loc; return t }
func (t Time) Location() *Location {
	if t.loc == nil {
		return UTC
	}
	return t.loc
}

func (t Time) Date() (int, Month, int) { y, m, d, _, _, _ := t.parts(); return y, Month(m), d }
func (t Time) Year() int               { y, _, _, _, _, _ := t.parts(); return y }
func (t Time) Month() Month            { _, m, _, _, _, _ := t.parts(); return Month(m) }
func (t Time) Day() int                { _, _, d, _, _, _ := t.parts(); return d }
func (t Time) Hour() int               { _, _, _, h, _, _ := t.parts(); return h }
func (t Time) Minute() int             { _, _, _, _, m, _ := t.parts(); return m }
func (t Time) Second() int             { _, _, _, _, _, s := t.parts(); return s }

func (t Time) parts() (int, int, int, int, int, int) {
	offset := 0
	if t.loc != nil {
		offset = t.loc.offset
	}
	sec := t.sec + int64(offset)
	days := floorDiv(sec, 86400)
	rem := sec - days*86400
	y, m, d := civilFromDays(days)
	return y, m, d, int(rem / 3600), int(rem / 60 % 60), int(rem % 60)
}

func (t Time) Format(layout string) string {
	if layout != RFC3339 && layout != RFC3339Nano {
		return ""
	}
	y, m, d, h, min, sec := t.parts()
	out := four(y) + "-" + two(m) + "-" + two(d) + "T" + two(h) + ":" + two(min) + ":" + two(sec)
	if layout == RFC3339Nano && t.nsec != 0 {
		out += "." + fraction9(int64(t.nsec))
	}
	offset := 0
	if t.loc != nil {
		offset = t.loc.offset
	}
	if offset == 0 {
		return out + "Z"
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return out + sign + two(offset/3600) + ":" + two(offset/60%60)
}

func (t Time) String() string { return t.Format(RFC3339Nano) }

func Parse(layout, value string) (Time, error) {
	if layout != RFC3339 && layout != RFC3339Nano {
		return Time{}, errors.New("time: unsupported layout")
	}
	if len(value) < 20 || value[4] != '-' || value[7] != '-' || value[10] != 'T' || value[13] != ':' || value[16] != ':' {
		return Time{}, errors.New("parsing time: invalid RFC3339 value")
	}
	y, ok := digits(value, 0, 4)
	if !ok {
		return Time{}, errors.New("parsing time: invalid year")
	}
	mo, ok := digits(value, 5, 7)
	if !ok {
		return Time{}, errors.New("parsing time: invalid month")
	}
	d, ok := digits(value, 8, 10)
	if !ok {
		return Time{}, errors.New("parsing time: invalid day")
	}
	h, ok := digits(value, 11, 13)
	if !ok {
		return Time{}, errors.New("parsing time: invalid hour")
	}
	mi, ok := digits(value, 14, 16)
	if !ok {
		return Time{}, errors.New("parsing time: invalid minute")
	}
	s, ok := digits(value, 17, 19)
	if !ok || mo < 1 || mo > 12 || d < 1 || d > daysInMonth(y, mo) || h > 23 || mi > 59 || s > 59 {
		return Time{}, errors.New("parsing time: value out of range")
	}
	i := 19
	ns := 0
	if i < len(value) && value[i] == '.' {
		i++
		start := i
		scale := 100000000
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			if i-start < 9 {
				ns += int(value[i]-'0') * scale
				scale /= 10
			}
			i++
		}
		if i == start {
			return Time{}, errors.New("parsing time: invalid fraction")
		}
	}
	offset := 0
	if i < len(value) && value[i] == 'Z' {
		i++
	} else if i+6 == len(value) && (value[i] == '+' || value[i] == '-') && value[i+3] == ':' {
		oh, a := digits(value, i+1, i+3)
		om, b := digits(value, i+4, i+6)
		if !a || !b || oh > 23 || om > 59 {
			return Time{}, errors.New("parsing time: invalid zone")
		}
		offset = oh*3600 + om*60
		if value[i] == '-' {
			offset = -offset
		}
		i += 6
	} else {
		return Time{}, errors.New("parsing time: invalid zone")
	}
	if i != len(value) {
		return Time{}, errors.New("parsing time: trailing text")
	}
	loc := UTC
	if offset != 0 {
		loc = &Location{name: "", offset: offset}
	}
	return Date(y, Month(mo), d, h, mi, s, ns, loc), nil
}

func daysFromCivil(y, m, d int) int64 {
	if m <= 2 {
		y--
	}
	era := floorDiv(int64(y), 400)
	yoe := int64(y) - era*400
	mp := m - 3
	if mp < 0 {
		mp += 12
	}
	doy := int64((153*mp+2)/5 + d - 1)
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}
func civilFromDays(z int64) (int, int, int) {
	z += 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := int(yoe + era*400)
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := int((5*doy + 2) / 153)
	d := int(doy) - ((153*mp + 2) / 5) + 1
	m := mp + 3
	if m > 12 {
		m -= 12
	}
	if m <= 2 {
		y++
	}
	return y, m, d
}
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b < 0 {
		q--
	}
	return q
}
func daysInMonth(y, m int) int {
	sizes := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if m == 2 && (y%4 == 0 && y%100 != 0 || y%400 == 0) {
		return 29
	}
	return sizes[m-1]
}
func digits(s string, a, b int) (int, bool) {
	if a < 0 || b > len(s) || a == b {
		return 0, false
	}
	n := 0
	for i := a; i < b; i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}
func two(v int) string { return string([]byte{byte('0' + v/10%10), byte('0' + v%10)}) }
func four(v int) string {
	return string([]byte{byte('0' + v/1000%10), byte('0' + v/100%10), byte('0' + v/10%10), byte('0' + v%10)})
}
