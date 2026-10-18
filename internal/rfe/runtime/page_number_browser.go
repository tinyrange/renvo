//go:build browser

package runtime

// The WASM emitter compares complete i64 tags for full-width address keys.
const nativePageNumberLimit = uint64(1) << 52
