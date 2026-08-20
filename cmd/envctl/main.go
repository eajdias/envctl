package main

import (
	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/ui/cli"
)

// Version is injected at build time via -ldflags "-X main.Version=$VERSION"
// (see the Makefile and .github/workflows/release.yml). Dev builds default to "dev".
var Version = "dev"

func main() {
	cli.InitApp(envctl.EmbeddedFS, Version)
	cli.Execute()
}
