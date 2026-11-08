package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic grava data em path de forma atômica: cria o diretório,
// escreve em um arquivo temporário no mesmo diretório, sincroniza e renomeia.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("criar diretório %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tyot-*.tmp")
	if err != nil {
		return fmt.Errorf("criar arquivo temporário em %q: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("escrever arquivo temporário: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sincronizar arquivo temporário: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fechar arquivo temporário: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("ajustar permissões do arquivo: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renomear arquivo para %q: %w", path, err)
	}
	return nil
}

// silenceStdout redireciona os.Stdout durante a extração para impedir que a
// biblioteca InnerTube (que imprime diagnósticos em stdout) polua o Markdown.
// Em modo quiet o destino é /dev/null; caso contrário, stderr.
func silenceStdout(quiet bool) func() {
	old := os.Stdout
	if quiet {
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err == nil {
			os.Stdout = devNull
			return func() {
				os.Stdout = old
				_ = devNull.Close()
			}
		}
	}
	os.Stdout = os.Stderr
	return func() {
		os.Stdout = old
	}
}
