package main

import (
	"embed"

	"github.com/construct-space/cli/cmd"
)

//go:embed all:templates/space
var templateFS embed.FS

func main() {
	cmd.TemplateFS = templateFS
	cmd.Execute()
}
