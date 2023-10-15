package assets

import "embed"

// Assets is the embedded assets objects for the included crd yaml data.
//
//go:embed crd/*.yaml
var Assets embed.FS
