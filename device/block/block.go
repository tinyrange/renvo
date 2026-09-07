// Package block defines synchronous 512-byte sector storage. Implementations
// must complete transfers before returning and must reject out-of-range I/O.
package block

const Size = 512

type Device interface {
	Blocks() uint32
	ReadBlocks(lba uint32, data []byte) error
	WriteBlocks(lba uint32, data []byte) error
	Sync() error
}

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrRange       Error = "block: invalid sector range"
	ErrTimeout     Error = "block: card timeout"
	ErrIO          Error = "block: transfer failed"
	ErrUnsupported Error = "block: unsupported card"
)

func Valid(total, lba uint32, size int) bool {
	return size >= 0 && size%Size == 0 && lba <= total && uint32(size/Size) <= total-lba
}
