//go:build !darwin && !windows

package assets

import (
	"os"
	"time"
)

func sourceCreatedAt(os.FileInfo) (time.Time, bool) { return time.Time{}, false }
