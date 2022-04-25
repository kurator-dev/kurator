package moreos

import "os"

const (
	// Exe is the runtime.GOOS-specific suffix for executables. Ex. "" unless windows which is ".exe"
	// See https://github.com/golang/go/issues/47567 for formalization
	Exe = exe
)

// IsExecutable returns true if the input can be run as an exec.Cmd
func IsExecutable(f os.FileInfo) bool {
	return isExecutable(f)
}
