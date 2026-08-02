//go:build tiny

package forms

// Tiny systems retain the Forms icon API but omit the optional designer
// artwork. Text-only controls require no embedded asset data.
var embeddedIconSet string
var embeddedControlIconMasks []byte
