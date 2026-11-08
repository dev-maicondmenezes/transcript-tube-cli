package cli

// Exit codes do tyot. Eles fazem parte do contrato observável do CLI e são
// documentados em docs/cli.md e na man page.
const (
	ExitOK         = 0
	ExitArgument   = 1
	ExitExtraction = 2
	ExitFileWrite  = 3
	ExitConfig     = 4
)

// codedError carrega um código de saída e uma mensagem já humanizada.
// silent evita imprimir novamente quando o comando já mostrou a ajuda.
type codedError struct {
	code   int
	err    error
	silent bool
}

func (e *codedError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *codedError) Unwrap() error {
	return e.err
}
