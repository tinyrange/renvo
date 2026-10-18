//go:build !browser

package runtime

// Native emitters use checked low-48-bit address keys.
const nativePageNumberLimit = uint64(1) << 36
