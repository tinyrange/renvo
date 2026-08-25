// Package board exposes the hardware capabilities of the board selected by the
// compilation target.
//
// Board targets add one build tag, such as m5nanoc6 or m5tab5, which selects
// the matching adapter in this package. There is intentionally no synthetic
// common interface: a program that asks for a capability its selected board
// does not provide fails to compile at that selector.
package board
