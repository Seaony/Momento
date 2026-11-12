//go:build !windows

package ui

import (
	"errors"
	"syscall"
)

func passwordDiskFull(err error) bool { return errors.Is(err, syscall.ENOSPC) }
