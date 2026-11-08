package cli

import (
	"os"
	"strings"
)

// EnvError é retornado por ValidateEnv quando um pré-requisito de ambiente
// está ausente. O código de saída associado é ExitExtraction (2).
type EnvError struct {
	Message string
}

func (e *EnvError) Error() string {
	return e.Message
}

const missingAPIKeyMessage = `erro: variável de ambiente YOUTUBE_API_KEY não está definida.

O tyot precisa da chave da YouTube Data API v3 para extrair metadados.
Defina-a antes de executar:

  export YOUTUBE_API_KEY="sua-chave"
  tyot <URL-ou-ID>

Como obter uma chave:
  1. Acesse https://console.cloud.google.com/apis/credentials
  2. Ative a YouTube Data API v3 no projeto.
  3. Crie uma credencial do tipo "Chave de API".
  4. Restrinja a chave à YouTube Data API v3.

A ajuda (tyot --help) e a versão (tyot --version) funcionam sem a chave.

Código de saída: 2`

// ValidateEnv verifica pré-requisitos de ambiente para execução real do CLI.
// Retorna nil quando OK ou um *EnvError com mensagem amigável.
func ValidateEnv(getenv func(string) string) error {
	if getenv == nil {
		getenv = os.Getenv
	}
	if strings.TrimSpace(getenv("YOUTUBE_API_KEY")) != "" {
		return nil
	}
	return &EnvError{Message: missingAPIKeyMessage}
}

// IsHelpType informa se a invocação é isenta da validação de YOUTUBE_API_KEY.
//
// São isentos: --help/-h, --version/-V, help, completion e combinações de
// flags curtas que contenham h ou V. Qualquer outra invocação — inclusive sem
// argumentos, com URL inválida ou com flag desconhecida — valida o ambiente
// primeiro.
func IsHelpType(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			break
		}
		switch arg {
		case "-h", "--help", "-V", "--version", "help", "completion":
			return true
		}
		if strings.HasPrefix(arg, "--help=") {
			return arg == "--help=true"
		}
		if strings.HasPrefix(arg, "--version=") {
			return arg == "--version=true"
		}
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			for _, r := range arg[1:] {
				if r == 'h' || r == 'V' {
					return true
				}
			}
		}
	}
	return false
}
