package config

import (
	"maps"
	"path"
)

func getNodeSelectorsByImage(
	imageName string,
	allSelectors map[string]map[string]string,
) map[string]string {
	nodeSelectors := make(map[string]string)

	for pattern, selectors := range allSelectors {
		match, err := path.Match(pattern, imageName)
		if err != nil || !match {
			continue
		}

		maps.Copy(nodeSelectors, selectors)

		break
	}

	defaultSelectors, defaultOk := allSelectors["default"]
	if defaultOk && len(nodeSelectors) == 0 {
		maps.Copy(nodeSelectors, defaultSelectors)
	}

	return nodeSelectors
}
