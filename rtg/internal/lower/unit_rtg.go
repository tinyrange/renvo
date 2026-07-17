//go:build rtg

package lower

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/check"
)

func discardCoreCheckStorage(info check.PackageInfo) {
	arena.Discard(info.CoreArenaStart, info.CoreArenaEnd)
}
