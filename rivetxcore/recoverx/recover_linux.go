//go:build linux
// +build linux

package recoverx

import (
	"os"
	"syscall"
)

// RedirectStderr to the file passed in
func RedirectStderr(path string) (err error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()
	err = syscall.Dup3(int(logFile.Fd()), int(os.Stderr.Fd()), 0)
	if err != nil {
		return
	}
	return
}
