package launcher

import "embed"

// Assets is the embedded asset objects for the clabernetes launcher.
//
//go:embed assets/*
var Assets embed.FS
