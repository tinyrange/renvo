//go:build !renvo

package graphics

func allocSurface() *Surface { return &Surface{} }
