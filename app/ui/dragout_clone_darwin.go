//go:build darwin

package ui

import (
	"errors"

	"golang.org/x/sys/unix"
)

// 用 APFS 写时复制建暂存文件：瞬间完成，改暂存文件不会动到原件。
// 跨卷、非 APFS 等不支持时返回 errCloneUnsupported，由调用方改为普通复制。
func cloneDragFile(source, destination string) error {
	err := unix.Clonefile(source, destination, unix.CLONE_NOFOLLOW)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EXDEV) || errors.Is(err, unix.ENOTTY) {
		return errCloneUnsupported
	}
	return err
}
