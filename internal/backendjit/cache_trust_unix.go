//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package backendjit

import (
	"os"
	"syscall"
)

func cachedFileOwnedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Geteuid()
}
