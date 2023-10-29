package util

import (
	"io"
	"net/http"
)

// WriteHTTPContentsFromPath writes content at the http path `path` into the writer w.
func WriteHTTPContentsFromPath(path string, w io.Writer) error {
	resp, err := http.Get(path) //nolint:gosec,noctx
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
