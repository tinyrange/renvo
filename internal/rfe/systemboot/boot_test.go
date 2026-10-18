package systemboot

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	hashcrc "hash/crc32"
	"testing"
)

func gzipAsset(t *testing.T, data []byte, level int) []byte {
	t.Helper()
	var b bytes.Buffer
	w, e := gzip.NewWriterLevel(&b, level)
	if e != nil {
		t.Fatal(e)
	}
	w.Name = "kernel"
	w.Comment = "boot asset"
	w.Extra = []byte{1, 2, 3, 4}
	if _, e = w.Write(data); e != nil {
		t.Fatal(e)
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestGunzipHostCompatibility(t *testing.T) {
	for _, n := range []int{0, 1, 31, 32769, 100000} {
		for _, level := range []int{-2, 0, 1, 9} {
			for _, pattern := range []int{0, 1, 2} {
				t.Run(fmt.Sprintf("%d/%d/%d", n, level, pattern), func(t *testing.T) {
					input := make([]byte, n)
					seed := uint32(1234)
					for j := range input {
						seed = seed*1664525 + 1013904223
						switch pattern {
						case 0:
							input[j] = 'A'
						case 1:
							input[j] = byte(j % 251)
						default:
							input[j] = byte(seed >> 24)
						}
					}
					data := gzipAsset(t, input, level)
					got, e := Gunzip(data, n)
					if e != nil || !bytes.Equal(got, input) {
						t.Fatal("round trip", e, len(got), len(input))
					}
					// Optional FHCRC covers all the optional header fields too.
					p := 12 + int(binary.LittleEndian.Uint16(data[10:]))
					for data[p] != 0 {
						p++
					}
					p++
					for data[p] != 0 {
						p++
					}
					p++
					head := append([]byte(nil), data[:p]...)
					head[3] |= 2
					sum := hashcrc.ChecksumIEEE(head)
					withCRC := append(head, byte(sum), byte(sum>>8))
					withCRC = append(withCRC, data[p:]...)
					got, e = Gunzip(withCRC, n)
					if e != nil || !bytes.Equal(got, input) {
						t.Fatal("header CRC", e)
					}
				})
			}
		}
	}
}
func TestGunzipRejectsCorruptionAndLimits(t *testing.T) {
	payload := bytes.Repeat([]byte("overlapping backreferences"), 4000)
	data := gzipAsset(t, payload, 9)
	cases := [][]byte{data[:0], data[:9], data[:len(data)-1], append(append([]byte(nil), data...), 0), append(append([]byte(nil), data...), data...)}
	for _, offset := range []int{0, 2, 3, len(data) - 8, len(data) - 4} {
		bad := append([]byte(nil), data...)
		bad[offset] ^= 0xff
		cases = append(cases, bad)
	}
	for j, bad := range cases {
		if _, e := Gunzip(bad, len(payload)); e == nil {
			t.Fatal("accepted corruption", j)
		}
	}
	if _, e := Gunzip(data, len(payload)-1); e == nil {
		t.Fatal("ignored output limit")
	}
	// A valid-checksummed empty stream must still contain a final block.
	empty := make([]byte, 18)
	copy(empty, []byte{31, 139, 8})
	if _, e := Gunzip(empty, 0); e == nil {
		t.Fatal("missing final block")
	}
	// Reserved block type with otherwise correct empty gzip trailer.
	reserved := append([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 0, 7}, make([]byte, 8)...)
	if _, e := Gunzip(reserved, 0); e == nil {
		t.Fatal("reserved block")
	}
}
func elfAsset() []byte {
	image := make([]byte, 260)
	copy(image, []byte{127, 'E', 'L', 'F', 2, 1, 1})
	b := binary.LittleEndian
	b.PutUint16(image[16:], 2)
	b.PutUint16(image[18:], 62)
	b.PutUint32(image[20:], 1)
	b.PutUint64(image[24:], 0xffff800000001002)
	b.PutUint64(image[32:], 64)
	b.PutUint16(image[52:], 64)
	b.PutUint16(image[54:], 56)
	b.PutUint16(image[56:], 1)
	p := image[64:]
	b.PutUint32(p, 1)
	b.PutUint64(p[8:], 256)
	b.PutUint64(p[16:], 0xffff800000001000)
	b.PutUint64(p[24:], 128)
	b.PutUint64(p[32:], 4)
	b.PutUint64(p[40:], 16)
	copy(image[256:], []byte{1, 2, 3, 4})
	return image
}
func TestLoadELFPhysicalEntryAndBSS(t *testing.T) {
	ram := bytes.Repeat([]byte{0xcc}, 512)
	entry, high, e := LoadELF64X86(elfAsset(), ram)
	if e != nil || entry != 130 || high != 144 || !bytes.Equal(ram[128:132], []byte{1, 2, 3, 4}) || !bytes.Equal(ram[132:144], make([]byte, 12)) || ram[127] != 0xcc || ram[144] != 0xcc {
		t.Fatal(entry, high, e, ram[127:145])
	}
}
func TestELFRejectsRangesWithoutWriting(t *testing.T) {
	for _, offset := range []int{4, 5, 18, 32, 54, 56, 64 + 8, 64 + 24 + 1, 64 + 32, 64 + 40 + 1} {
		image := elfAsset()
		image[offset] = 0xff
		ram := bytes.Repeat([]byte{0xcc}, 512)
		before := append([]byte(nil), ram...)
		if _, _, e := LoadELF64X86(image, ram); e == nil {
			t.Fatal("accepted malformed ELF", offset)
		}
		if !bytes.Equal(ram, before) {
			t.Fatal("partial invalid load", offset)
		}
	}
	for n := 0; n < 120; n++ {
		if _, _, e := LoadELF64X86(elfAsset()[:n], make([]byte, 512)); e == nil {
			t.Fatal("accepted truncation", n)
		}
	}
}
func TestKernelELFPayloadBounds(t *testing.T) {
	elf := elfAsset()
	gz := gzipAsset(t, elf, 9)
	kernel := make([]byte, 4096+len(gz))
	copy(kernel[0x202:], "HdrS")
	kernel[0x1f1] = 7
	binary.LittleEndian.PutUint32(kernel[0x24c:], uint32(len(gz)))
	copy(kernel[4096:], gz)
	image, header, e := KernelELF(kernel, 1024)
	if e != nil || !bytes.Equal(image, elf) || len(header) != 4096 {
		t.Fatal("bzImage", e)
	}
	binary.LittleEndian.PutUint32(kernel[0x248:], 0xffffffff)
	if _, _, e := KernelELF(kernel, 1024); e == nil {
		t.Fatal("payload offset")
	}
	if _, _, e := KernelELF(kernel[:0x269], 1024); e == nil {
		t.Fatal("setup truncation")
	}
}
