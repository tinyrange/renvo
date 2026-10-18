// Package systemboot loads boot assets without host compression or ELF dependencies.
package systemboot

import (
	"encoding/binary"
	"fmt"
)

// Gunzip decodes one RFC 1952 member, including its checksum. It intentionally
// rejects concatenated members and trailing data: a kernel payload is one asset.
// The declared output size is checked before allocation and enforced on every
// literal and match, so corrupt assets cannot grow the output arena unboundedly.
func Gunzip(data []byte, limit int) ([]byte, error) {
	if len(data) < 18 || data[0] != 31 || data[1] != 139 || data[2] != 8 || data[3]&0xe0 != 0 {
		return nil, fmt.Errorf("invalid gzip header")
	}
	end := len(data) - 8
	p := 10
	flags := data[3]
	if flags&4 != 0 {
		if p+2 > end {
			return nil, fmt.Errorf("truncated gzip extra")
		}
		n := int(binary.LittleEndian.Uint16(data[p:]))
		p += 2
		if n > end-p {
			return nil, fmt.Errorf("truncated gzip extra")
		}
		p += n
	}
	for _, flag := range [2]byte{8, 16} {
		if flags&flag != 0 {
			for p < end && data[p] != 0 {
				p++
			}
			if p >= end {
				return nil, fmt.Errorf("unterminated gzip name/comment")
			}
			p++
		}
	}
	if flags&2 != 0 {
		if p+2 > end || uint16(crc32(data[:p])) != binary.LittleEndian.Uint16(data[p:]) {
			return nil, fmt.Errorf("invalid gzip header checksum")
		}
		p += 2
	}
	size := uint64(binary.LittleEndian.Uint32(data[end+4:]))
	if limit < 0 || size > uint64(limit) {
		return nil, fmt.Errorf("gzip output exceeds limit")
	}
	r := bitReader{data: data[p:end]}
	out := make([]byte, 0, int(size))
	var lit, dist, codes huffman
	var lengths [320]uint8
	final := false
	for !final && !r.bad {
		final = r.take(1) != 0
		kind := r.take(2)
		if kind == 0 {
			r.align()
			n := int(r.take(16))
			inverse := r.take(16)
			if r.bad || uint32(n)^inverse != 65535 || n > cap(out)-len(out) {
				return nil, fmt.Errorf("invalid stored deflate block")
			}
			for j := 0; j < n; j++ {
				out = append(out, byte(r.take(8)))
			}
			continue
		}
		if kind == 3 {
			return nil, fmt.Errorf("reserved deflate block type")
		}
		clear(lengths[:])
		if kind == 1 {
			for j := 0; j < 288; j++ {
				switch {
				case j < 144:
					lengths[j] = 8
				case j < 256:
					lengths[j] = 9
				case j < 280:
					lengths[j] = 7
				default:
					lengths[j] = 8
				}
			}
			if !lit.build(lengths[:288], false) {
				return nil, fmt.Errorf("invalid fixed literals")
			}
			for j := 0; j < 32; j++ {
				lengths[j] = 5
			}
			if !dist.build(lengths[:32], false) {
				return nil, fmt.Errorf("invalid fixed distances")
			}
		} else {
			nl := 257 + int(r.take(5))
			nd := 1 + int(r.take(5))
			nc := 4 + int(r.take(4))
			if nl > 286 {
				return nil, fmt.Errorf("invalid dynamic literal count")
			}
			order := [19]int{16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15}
			for j := 0; j < nc; j++ {
				lengths[order[j]] = uint8(r.take(3))
			}
			if r.bad || !codes.build(lengths[:19], false) || codes.width == 0 {
				return nil, fmt.Errorf("invalid code-length tree")
			}
			clear(lengths[:])
			for j := 0; j < nl+nd && !r.bad; {
				s := r.symbol(&codes)
				if s < 16 {
					lengths[j] = uint8(s)
					j++
					continue
				}
				n := 0
				value := uint8(0)
				switch s {
				case 16:
					if j == 0 {
						return nil, fmt.Errorf("missing previous code length")
					}
					value = lengths[j-1]
					n = 3 + int(r.take(2))
				case 17:
					n = 3 + int(r.take(3))
				case 18:
					n = 11 + int(r.take(7))
				default:
					return nil, fmt.Errorf("invalid code-length symbol")
				}
				if n > nl+nd-j {
					return nil, fmt.Errorf("code-length repeat overflow")
				}
				for k := 0; k < n; k++ {
					lengths[j] = value
					j++
				}
			}
			if r.bad || lengths[256] == 0 || !lit.build(lengths[:nl], true) || !dist.build(lengths[nl:nl+nd], true) {
				return nil, fmt.Errorf("invalid dynamic trees")
			}
		}
		for !r.bad {
			s := r.symbol(&lit)
			if s < 256 {
				if len(out) == cap(out) {
					return nil, fmt.Errorf("gzip output exceeds declared size")
				}
				out = append(out, byte(s))
				continue
			}
			if s == 256 {
				break
			}
			if s > 285 {
				return nil, fmt.Errorf("invalid deflate length")
			}
			k := s - 257
			n := int(lengthBase[k]) + int(r.take(uint(lengthExtra[k])))
			d := r.symbol(&dist)
			if r.bad || d >= 30 {
				return nil, fmt.Errorf("invalid deflate distance")
			}
			back := int(distanceBase[d]) + int(r.take(uint(distanceExtra[d])))
			if r.bad || back > len(out) || n > cap(out)-len(out) {
				return nil, fmt.Errorf("invalid deflate match range")
			}
			// Bytewise copy is required: matches may overlap, including distance one.
			for j := 0; j < n; j++ {
				out = append(out, out[len(out)-back])
			}
		}
	}
	r.align()
	if r.bad || r.pos-int(r.count/8) != len(r.data) || len(out) != int(size) {
		return nil, fmt.Errorf("truncated or trailing gzip payload")
	}
	if crc32(out) != binary.LittleEndian.Uint32(data[end:]) {
		return nil, fmt.Errorf("gzip checksum mismatch")
	}
	return out, nil
}

type bitReader struct {
	data  []byte
	pos   int
	bits  uint32
	count uint
	bad   bool
}

func (r *bitReader) fill(n uint) {
	for r.count < n && r.pos < len(r.data) {
		r.bits |= uint32(r.data[r.pos]) << r.count
		r.count += 8
		r.pos++
	}
}
func (r *bitReader) take(n uint) uint32 {
	r.fill(n)
	if r.count < n {
		r.bad = true
		return 0
	}
	v := r.bits & ((uint32(1) << n) - 1)
	r.bits >>= n
	r.count -= n
	return v
}
func (r *bitReader) align() { n := r.count & 7; r.bits >>= n; r.count -= n }
func (r *bitReader) symbol(h *huffman) int {
	r.fill(h.width)
	e := h.table[r.bits&((uint32(1)<<h.width)-1)]
	n := uint(e >> 9)
	if n == 0 || n > r.count {
		r.bad = true
		return 0
	}
	r.bits >>= n
	r.count -= n
	return int(e & 511)
}

type huffman struct {
	table [32768]uint16
	width uint
}

// build constructs canonical codes and reverses them for the LSB-first stream.
// Only literal/distance alphabets permit the single-symbol incomplete tree;
// an empty distance alphabet is legal until a match actually uses it.
func (h *huffman) build(lengths []uint8, single bool) bool {
	var count, next [16]int
	h.width = 0
	for _, n := range lengths {
		if n > 15 {
			return false
		}
		if n != 0 {
			count[n]++
			if uint(n) > h.width {
				h.width = uint(n)
			}
		}
	}
	clear(h.table[:1<<h.width])
	if h.width == 0 {
		return single
	}
	left := 1
	for n := 1; n <= 15; n++ {
		left = left*2 - count[n]
		if left < 0 {
			return false
		}
	}
	if left != 0 && !(single && h.width == 1) {
		return false
	}
	code := 0
	for n := 1; n <= 15; n++ {
		code = (code + count[n-1]) << 1
		next[n] = code
	}
	for symbol, n := range lengths {
		if n == 0 {
			continue
		}
		c := next[n]
		next[n]++
		reverse := 0
		for j := uint8(0); j < n; j++ {
			reverse = reverse<<1 | c&1
			c >>= 1
		}
		e := uint16(n)<<9 | uint16(symbol)
		for j := reverse; j < 1<<h.width; j += 1 << n {
			h.table[j] = e
		}
	}
	return true
}

var lengthBase = [29]uint16{3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31, 35, 43, 51, 59, 67, 83, 99, 115, 131, 163, 195, 227, 258}
var lengthExtra = [29]uint8{0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0}
var distanceBase = [30]uint16{1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193, 257, 385, 513, 769, 1025, 1537, 2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577}
var distanceExtra = [30]uint8{0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13}

func crc32(data []byte) uint32 {
	var table [256]uint32
	for j := 0; j < 256; j++ {
		c := uint32(j)
		for k := 0; k < 8; k++ {
			if c&1 != 0 {
				c = c>>1 ^ 0xedb88320
			} else {
				c >>= 1
			}
		}
		table[j] = c
	}
	c := ^uint32(0)
	for _, b := range data {
		c = table[byte(c)^b] ^ c>>8
	}
	return ^c
}
