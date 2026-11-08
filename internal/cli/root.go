package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dev-maicondmenezes/transcript-tube-cli/internal/config"
	core "github.com/dev-maicondmenezes/transcript-tube-core/core"
)

// Extractor é o seam de teste do CLI. A implementação real delega ao core.
type Extractor interface {
	Extract(ctx context.Context, videoID string, opts core.Options) (*core.VideoData, error)
}

type coreExtractor struct{}

func (coreExtractor) Extract(ctx context.Context, videoID string, opts core.Options) (*core.VideoData, error) {
	return core.ExtractWithOptions(ctx, videoID, opts)
}

// App agrupa as dependências de I/O e de extração para permitir testes
// offline com um fake.
type App struct {
	In                io.Reader
	Out               io.Writer
	Err               io.Writer
	InIsTTY           bool
	Getenv            func(string) string
	Extractor         Extractor
	SilenceCoreStdout bool
	Now               func() time.Time
}

// NewApp cria um App com as dependências reais de produção.
func NewApp(in io.Reader, out, err io.Writer) *App {
	return &App{
		In:        in,
		Out:       out,
		Err:       err,
		Getenv:    os.Getenv,
		Extractor: coreExtractor{},
		Now:       time.Now,
	}
}

// Run é o entrypoint usado por cmd/tyot/main.go.
func Run(args []string, stdout, stderr io.Writer) int {
	app := NewApp(os.Stdin, stdout, stderr)
	app.InIsTTY = isTerminal(os.Stdin)
	app.SilenceCoreStdout = true
	return app.Run(args)
}

// Run executa o CLI e devolve o código de saída.
func (a *App) Run(args []string) int {
	if a.Out == nil {
		a.Out = io.Discard
	}
	if a.Err == nil {
		a.Err = io.Discard
	}
	if a.Getenv == nil {
		a.Getenv = os.Getenv
	}
	if a.Extractor == nil {
		a.Extractor = coreExtractor{}
	}
	if a.Now == nil {
		a.Now = time.Now
	}

	// A validação acontece antes do parsing do Cobra para que flags
	// desconhecidas e argumentos inválidos também falhem primeiro por causa
	// da env ausente (T2.11/T2.12).
	if !IsHelpType(args) {
		if err := ValidateEnv(a.Getenv); err != nil {
			printError(a.Err, err)
			return ExitExtraction
		}
	}

	cmd := newRootCommand(a)
	cmd.SetArgs(args)
	cmd.SetOut(a.Out)
	cmd.SetErr(a.Err)
	if err := cmd.Execute(); err != nil {
		return a.handleError(err)
	}
	return ExitOK
}

func (a *App) handleError(err error) int {
	var coded *codedError
	if errors.As(err, &coded) {
		if !coded.silent {
			printError(a.Err, coded.err)
		}
		return coded.code
	}
	var envErr *EnvError
	if errors.As(err, &envErr) {
		printError(a.Err, envErr)
		return ExitExtraction
	}
	var cfgErr *config.Error
	if errors.As(err, &cfgErr) {
		printError(a.Err, cfgErr)
		return ExitConfig
	}
	printError(a.Err, err)
	return ExitArgument
}

func printError(w io.Writer, err error) {
	if err == nil {
		return
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return
	}
	if strings.HasPrefix(message, "erro:") {
		fmt.Fprintln(w, message)
		return
	}
	fmt.Fprintf(w, "erro: %s\n", message)
}

func newRootCommand(app *App) *cobra.Command {
	flags := &rootFlags{}
	cmd := &cobra.Command{
		Use:   "tyot URL-ou-ID",
		Short: "Extrai metadados e transcrição do YouTube para Markdown",
		Long: `tyot — extrai metadados e transcrição do YouTube para Markdown

O comando aceita uma URL do YouTube ou um ID de 11 caracteres e escreve
Markdown em stdout (ou no arquivo indicado por --output).`,
		Example: `  tyot dQw4w9WgXcQ
  tyot "https://youtu.be/dQw4w9WgXcQ" -o out/rick.md
  tyot "https://www.youtube.com/watch?v=dQw4w9WgXcQ" --lang en`,
		Args: func(cmd *cobra.Command, args []string) error {
			if flags.version {
				return nil
			}
			return cobra.MaximumNArgs(1)(cmd, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.version {
				printVersion(cmd.OutOrStdout())
				return nil
			}
			return app.runExtract(cmd, args, flags)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flags.version || cmd.Name() == "completion" || cmd.Name() == "help" {
				return nil
			}
			return ValidateEnv(app.Getenv)
		},
	}

	cmd.SetUsageTemplate(usageTemplate)
	cmd.SetHelpTemplate(helpTemplate)
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%s", translateFlagError(err.Error()))
	})
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.Flags().SortFlags = false

	cmd.Flags().StringVarP(&flags.lang, "lang", "l", "pt", "Idioma preferido da transcrição")
	cmd.Flags().StringVarP(&flags.output, "output", "o", "", "Arquivo de saída (padrão: stdout)")
	cmd.Flags().DurationVar(&flags.timeout, "timeout", 60*time.Second, "Timeout total da extração")
	cmd.Flags().BoolVar(&flags.allowPartial, "allow-partial", false, "Retorna Markdown parcial se a transcrição falhar")
	cmd.Flags().BoolVar(&flags.noChapters, "no-chapters", false, "Desabilita headings de capítulos no Markdown")
	cmd.Flags().BoolVarP(&flags.quiet, "quiet", "q", false, "Suprime logs de progresso em stderr")
	cmd.Flags().StringVar(&flags.configPath, "config", "", "Arquivo de configuração (padrão: $XDG_CONFIG_HOME/tyot/config.yaml)")
	cmd.Flags().BoolVarP(&flags.version, "version", "V", false, "Mostra versão, commit e data de build")
	cmd.Flags().BoolP("help", "h", false, "Mostra esta ajuda")

	cmd.AddCommand(newCompletionCommand(cmd))
	return cmd
}

func translateFlagError(message string) string {
	if strings.HasPrefix(message, "unknown flag:") {
		remainder := strings.TrimSpace(strings.TrimPrefix(message, "unknown flag:"))
		return "flag desconhecida: " + remainder
	}
	if strings.HasPrefix(message, "unknown shorthand flag:") {
		remainder := strings.TrimPrefix(message, "unknown shorthand flag:")
		return "flag curta desconhecida:" + strings.TrimSpace(remainder)
	}
	return message
}

// NewRootCommandForDocs expõe a árvore de comandos para cmd/gen-man sem
// executar validação de ambiente ou extração.
func NewRootCommandForDocs() *cobra.Command {
	app := &App{
		Out:       io.Discard,
		Err:       io.Discard,
		Getenv:    func(string) string { return "" },
		Extractor: coreExtractor{},
		Now:       time.Now,
	}
	return newRootCommand(app)
}

type rootFlags struct {
	lang         string
	output       string
	timeout      time.Duration
	allowPartial bool
	noChapters   bool
	quiet        bool
	configPath   string
	version      bool
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const usageTemplate = `Uso:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if gt (len .Aliases) 0}}

Atalhos:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Exemplos:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Comandos disponíveis:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Flags globais:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [comando] --help" para mais informações sobre um comando.{{end}}
`

const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
Requisitos:
  Variável de ambiente YOUTUBE_API_KEY definida.
  Como obter: https://console.cloud.google.com/apis/credentials

Documentação completa: man tyot
Reportar bugs: https://github.com/dev-maicondmenezes/transcript-tube-cli/issues
`
