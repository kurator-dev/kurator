//go:build !windows

package moreos

import "os"

const (
	exe = ""
)

func isExecutable(f os.FileInfo) bool {
	return f.Mode()&0o111 != 0
}
