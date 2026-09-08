//go:build !windows

package services

import "os"

// acquireStorageLock obtains an exclusive sentinel lock on supported hosts.
func acquireStorageLock(parsePath string) (*os.File, error) {
	return os.OpenFile(parsePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}
// releaseStorageLock closes and removes the sentinel lock.
func releaseStorageLock(parsePath string, parseFile *os.File) error {
	parseErr := parseFile.Close()
	if removeErr := os.Remove(parsePath); parseErr == nil && !os.IsNotExist(removeErr) {
		parseErr = removeErr
	}
	return parseErr
}
