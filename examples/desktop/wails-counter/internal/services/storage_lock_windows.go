//go:build windows

package services

import (
	"golang.org/x/sys/windows"
	"os"
)

// acquireStorageLock obtains an exclusive OS-level ownership handle.
func acquireStorageLock(parsePath string) (*os.File, error) {
	parseHandle, parseErr := windows.CreateFile(windows.StringToUTF16Ptr(parsePath), windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if parseErr != nil {
		return nil, parseErr
	}
	return os.NewFile(uintptr(parseHandle), parsePath), nil
}

// releaseStorageLock releases the OS-level ownership handle.
func releaseStorageLock(parsePath string, parseFile *os.File) error { return parseFile.Close() }
