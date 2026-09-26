//go:build renvo

package errors

// Replaced by type-assertion dispatch in the linker. This grants no field
// reflection capability and does not depend on //renvo:reflect annotations.
func asTarget(err error, target any) (bool, bool) { return false, false }
