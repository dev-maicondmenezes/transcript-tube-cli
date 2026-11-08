package cli

import (
	"fmt"
	"io"
	"runtime"
)

// Estes valores são sobrescritos pelo GoReleaser via -ldflags -X.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "tyot %s\n", Version)
	fmt.Fprintf(w, "commit: %s\n", Commit)
	fmt.Fprintf(w, "built: %s\n", Date)
	fmt.Fprintf(w, "go: %s\n", runtime.Version())
}
