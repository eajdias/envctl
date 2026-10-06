package envctl

import (
	"embed"
)

// EmbeddedFS embeds the manifests and configs directories into the standalone
// binary. The `all:` prefix is required, not cosmetic: plain //go:embed skips
// files starting with `_`, which would drop configs/git/hooks/_envctl-delegate
// and silently break the deployed git hook chain.
//
//go:embed all:manifests all:configs
var EmbeddedFS embed.FS
