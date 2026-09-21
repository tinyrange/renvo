package main

import (
	"bytes"
	"encoding/base64"
	"hash/fnv"
	"io"
	"unicode/utf8"
)

func main() {
	r := bytes.NewReader([]byte("aéZ"))
	b, err := r.ReadByte()
	if b != 'a' || err != nil {
		panic("byte")
	}
	ch, n, err := r.ReadRune()
	if ch != 'é' || n != 2 || err != nil {
		panic("rune")
	}
	if r.UnreadRune() != nil || r.Len() != 3 {
		panic("unread")
	}
	pos, err := r.Seek(-1, io.SeekEnd)
	if pos != 3 || err != nil {
		panic("seek")
	}
	b, err = r.ReadByte()
	if b != 'Z' || err != nil {
		panic("tail")
	}
	if _, err = r.ReadByte(); err != io.EOF {
		panic("eof")
	}
	if !utf8.Valid([]byte("aé")) || utf8.Valid([]byte("\xff")) {
		panic("valid")
	}
	ch, n = utf8.DecodeLastRune([]byte("aé"))
	if ch != 'é' || n != 2 {
		panic("last")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("foobar"))
	if encoded != "Zm9vYmFy" {
		panic("encode")
	}
	decoded, err := base64.StdEncoding.DecodeString("Z\r\ng==")
	if string(decoded) != "f" || err != nil {
		panic("decode")
	}
	if _, err = base64.StdEncoding.Strict().DecodeString("Zh=="); err == nil {
		panic("strict")
	}
	h := fnv.New32a()
	h.Write([]byte("hello"))
	if h.Sum32() != 0x4f9f2cab {
		panic("fnv32")
	}
	h64 := fnv.New64a()
	h64.Write([]byte("hello"))
	if h64.Sum64() != 0xa430d84680aabd0b {
		panic("fnv64")
	}
	print("PASS\n")
}
