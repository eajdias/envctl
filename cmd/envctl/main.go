package main

import (
	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/ui/cli"
)

func main() {
	cli.InitApp(envctl.EmbeddedFS)
	cli.Execute()
}
