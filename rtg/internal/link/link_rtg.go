//go:build rtg

package link

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/build"
)

func discardLinkedPackageUnit(pkg build.PackageUnit) {
	arena.Discard(pkg.ArenaStart, pkg.ArenaEnd)
}
