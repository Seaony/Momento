//go:build unix

package media

import (
	"runtime"
	"syscall"
)

// maxRSS 返回进程到目前为止的峰值常驻内存（字节）。
func maxRSS() int64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	// macOS 上单位是字节，Linux 上是 KB
	if runtime.GOOS == "darwin" {
		return int64(ru.Maxrss)
	}
	return int64(ru.Maxrss) * 1024
}
