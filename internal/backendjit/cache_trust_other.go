//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package backendjit

import "os"

func cachedFileOwnedByCurrentUser(info os.FileInfo) bool {
	return false
}
