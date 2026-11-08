// Command gen-man gera docs/man/tyot.1 e os scripts de completion a partir da
// árvore de comandos do Cobra. Não é o binário de usuário e não valida
// YOUTUBE_API_KEY.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen-man: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := cli.NewRootCommandForDocs()
	root.DisableAutoGenTag = true

	if err := os.MkdirAll("docs/man", 0o755); err != nil {
		return err
	}
	header := &doc.GenManHeader{
		Title:   "TYOT",
		Section: "1",
		Date:    buildDate(),
		Source:  "tyot",
		Manual:  "tyot Manual",
	}
	pages := []struct {
		filename string
		command  *cobra.Command
	}{
		{filename: "tyot.1", command: root},
	}
	for _, subcommand := range root.Commands() {
		if !subcommand.IsAvailableCommand() || subcommand.Name() == "help" {
			continue
		}
		subcommand.DisableAutoGenTag = true
		pages = append(pages, struct {
			filename string
			command  *cobra.Command
		}{
			filename: "tyot-" + subcommand.Name() + ".1",
			command:  subcommand,
		})
	}
	for _, page := range pages {
		manFile, err := os.Create(filepath.Join("docs/man", page.filename))
		if err != nil {
			return err
		}
		if err := doc.GenMan(page.command, header, manFile); err != nil {
			_ = manFile.Close()
			return err
		}
		if err := manFile.Close(); err != nil {
			return err
		}
	}

	if err := os.MkdirAll("docs/completions", 0o755); err != nil {
		return err
	}
	completions := []struct {
		path string
		gen  func(io.Writer) error
	}{
		{"docs/completions/tyot.bash", func(w io.Writer) error { return root.GenBashCompletionV2(w, true) }},
		{"docs/completions/tyot.zsh", func(w io.Writer) error { return root.GenZshCompletion(w) }},
		{"docs/completions/_tyot", func(w io.Writer) error { return root.GenZshCompletion(w) }},
		{"docs/completions/tyot.fish", func(w io.Writer) error { return root.GenFishCompletion(w, true) }},
	}
	for _, completion := range completions {
		if err := writeCompletion(completion.path, completion.gen); err != nil {
			return err
		}
	}

	fmt.Println("gerado: docs/man/tyot.1 e docs/completions/*")
	return nil
}

func writeCompletion(path string, gen func(io.Writer) error) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gen(file); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func buildDate() *time.Time {
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		if seconds, err := strconv.ParseInt(epoch, 10, 64); err == nil {
			date := time.Unix(seconds, 0).UTC()
			return &date
		}
	}
	date := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	return &date
}
