//go:build !windows

package browser

import (
	"errors"
	"syscall"
)

func isAddressInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}
