package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "out.md")
	data := []byte("# hello\n")
	if err := writeFileAtomic(path, data); err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content = %q, want %q", got, data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %o, want 644", info.Mode().Perm())
	}
}

func TestWriteFileAtomicInvalidParent(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := writeFileAtomic(filepath.Join(blocker, "out.md"), []byte("x")); err == nil {
		t.Fatal("writeFileAtomic() error = nil, want error")
	}
}

func TestSilenceStdoutQuietDiscardsWrites(t *testing.T) {
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = old })

	restore := silenceStdout(true)
	fmt.Println("ruído")
	restore()
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("captured %q, want empty", data)
	}
}
