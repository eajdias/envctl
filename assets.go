package envctl

import (
	"embed"
)

// EmbeddedFS embeds the entire manifests and configs directories into the standalone binary.
//
//go:embed all:manifests all:configs
var EmbeddedFS embed.FS
