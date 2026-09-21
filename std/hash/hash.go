// Package hash provides interfaces for incremental hash functions.
package hash

type Hash interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
	Reset()
	Size() int
	BlockSize() int
}
type Hash32 interface {
	Hash
	Sum32() uint32
}
type Hash64 interface {
	Hash
	Sum64() uint64
}
