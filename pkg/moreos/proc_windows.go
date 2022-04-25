//go:build windows

package moreos

import (
	"os"
	"strings"
)

const (
	exe = ".exe"
)

func isExecutable(f os.FileInfo) bool {
	return strings.HasSuffix(f.Name(), exe)
}
