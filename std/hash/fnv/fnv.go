// Package fnv implements the non-cryptographic FNV-1 and FNV-1a hashes.
package fnv

import "hash"

type digest32 struct {
	value     uint32
	alternate bool
}
type digest64 struct {
	value     uint64
	alternate bool
}

func New32() hash.Hash32  { return &digest32{value: 2166136261} }
func New32a() hash.Hash32 { return &digest32{value: 2166136261, alternate: true} }
func New64() hash.Hash64  { return &digest64{value: 14695981039346656037} }
func New64a() hash.Hash64 { return &digest64{value: 14695981039346656037, alternate: true} }

func (d *digest32) Reset()         { d.value = 2166136261 }
func (d *digest32) Size() int      { return 4 }
func (d *digest32) BlockSize() int { return 1 }
func (d *digest32) Sum32() uint32  { return d.value }
func (d *digest32) Write(p []byte) (int, error) {
	for _, b := range p {
		if d.alternate {
			d.value ^= uint32(b)
		}
		d.value *= 16777619
		if !d.alternate {
			d.value ^= uint32(b)
		}
	}
	return len(p), nil
}
func (d *digest32) Sum(dst []byte) []byte {
	n := d.value
	return append(dst, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
func (d *digest64) Reset()         { d.value = 14695981039346656037 }
func (d *digest64) Size() int      { return 8 }
func (d *digest64) BlockSize() int { return 1 }
func (d *digest64) Sum64() uint64  { return d.value }
func (d *digest64) Write(p []byte) (int, error) {
	for _, b := range p {
		if d.alternate {
			d.value ^= uint64(b)
		}
		d.value *= 1099511628211
		if !d.alternate {
			d.value ^= uint64(b)
		}
	}
	return len(p), nil
}
func (d *digest64) Sum(dst []byte) []byte {
	n := d.value
	return append(dst, byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32), byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
