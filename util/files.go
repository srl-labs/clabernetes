package util

import (
	"errors"
	"io/fs"
	"os"
)

// MustCreateDirectory creates a directory at path `directory` with provided permissions, it panics
// if an error is encountered.
func MustCreateDirectory(directory string, permissions fs.FileMode) {
	err := os.MkdirAll(directory, permissions)
	if err != nil {
		panic(err)
	}
}

// MustFileExists returns true if a given file exists, otherwise false, it panics if any error is
// encountered.
func MustFileExists(f string) bool {
	_, err := os.Stat(f)

	return !errors.Is(err, os.ErrNotExist)
}
