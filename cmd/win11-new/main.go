package main

import (
	win11new "github.com/eajdias/win11-new"
	"github.com/eajdias/win11-new/internal/ui/cli"
)

func main() {
	cli.InitApp(win11new.EmbeddedFS)
	cli.Execute()
}
