package util

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
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

// ResolveAtFileOrURL returns the bytes from `path` where path is either a filepath or URL.
func ResolveAtFileOrURL(path string) ([]byte, error) {
	var b []byte

	switch {
	case strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://"):
		resp, err := http.Get(path) //nolint:gosec,noctx
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close() //nolint

		b, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

	default: // fall-through to local filesystem
		var err error

		b, err = os.ReadFile(path) //nolint:gosec
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}
