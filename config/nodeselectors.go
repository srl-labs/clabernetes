package config

import (
	"maps"
	"path"
)

// GetNodeSelectorsByImage returns the node selectors to be applied to support the image provided.
func GetNodeSelectorsByImage(
	imageName string,
	allSelectors map[string]map[string]string,
) map[string]string {
	longestPattern := 0
	nodeSelectors := make(map[string]string)

	for pattern, selectors := range allSelectors {
		match, err := path.Match(pattern, imageName)
		if err != nil || !match {
			continue
		}

		// Choose the most specific match (longest pattern)
		if len(pattern) > longestPattern {
			longestPattern = len(pattern)

			maps.Copy(nodeSelectors, selectors)
		}
	}

	defaultSelectors, defaultOk := allSelectors["default"]
	if defaultOk && len(nodeSelectors) == 0 {
		maps.Copy(nodeSelectors, defaultSelectors)
	}

	return nodeSelectors
}
