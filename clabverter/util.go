package clabverter

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
)

func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func loadContentAtURL(path string) ([]byte, error) {
	resp, err := http.Get(path) //nolint:gosec,noctx
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() //nolint

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func gitHubNormalToRawLink(path string) string {
	p := regexp.MustCompile(
		`(?mi)https?:\/\/(?:www\.)?github\.com\/(?P<GroupRepo>.*?\/.*?)\/(?:(blob)|(tree))(?P<Path>.*)`, //nolint:lll
	)

	paramsMap := clabernetesutil.RegexStringSubMatchToMap(p, path)

	return fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s", paramsMap["GroupRepo"], paramsMap["Path"],
	)
}
