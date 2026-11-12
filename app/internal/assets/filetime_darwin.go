package assets

import (
	"os"
	"syscall"
	"time"
)

func sourceCreatedAt(info os.FileInfo) (time.Time, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Birthtimespec.Sec == 0 {
		return time.Time{}, false
	}
	return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec), true
}
