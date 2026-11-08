package main

import (
	"os"

	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
