// scanner_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestDir cria um diretório temporário com arquivos para testar.
func createTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Arquivo Go com 3 linhas
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)

	// Arquivo markdown com 3 linhas
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Título\n\nTexto\n"), 0644)

	// Arquivo sem extensão com 1 linha
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n"), 0644)

	// Subdiretório normal
	subDir := filepath.Join(dir, "pkg")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "util.go"), []byte("package pkg\n"), 0644)

	// Arquivo binário (contém byte nulo)
	os.WriteFile(filepath.Join(dir, "binary.bin"), []byte{0x00, 0x01, 0x02}, 0644)

	// Diretório oculto (deve ser ignorado)
	hiddenDir := filepath.Join(dir, ".git")
	os.Mkdir(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "config"), []byte("ignored\n"), 0644)

	return dir
}

func TestScan_CountsFilesAndDirs(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir)

	// main.go, README.md, Makefile, binary.bin, pkg/util.go = 5 arquivos
	if stats.Files != 5 {
		t.Errorf("esperado 5 arquivos, obtido %d", stats.Files)
	}

	// pkg/ = 1 diretório (ocultos ignorados)
	if stats.Dirs != 1 {
		t.Errorf("esperado 1 diretório, obtido %d", stats.Dirs)
	}
}

func TestScan_CountsLines(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir)

	// main.go=3, README.md=3, Makefile=1, pkg/util.go=1, binary.bin=0 → 8 linhas
	if stats.Lines != 8 {
		t.Errorf("esperado 8 linhas, obtido %d", stats.Lines)
	}
}

func TestScan_GroupsByExtension(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir)

	// .go: main.go(3) + pkg/util.go(1) = 4
	if stats.ByExt[".go"] != 4 {
		t.Errorf("esperado .go=4, obtido %d", stats.ByExt[".go"])
	}

	// .md: README.md(3)
	if stats.ByExt[".md"] != 3 {
		t.Errorf("esperado .md=3, obtido %d", stats.ByExt[".md"])
	}

	// sem ext: Makefile(1)
	if stats.ByExt["(sem ext)"] != 1 {
		t.Errorf("esperado (sem ext)=1, obtido %d", stats.ByExt["(sem ext)"])
	}
}

func TestScan_IgnoresHiddenEntries(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir)

	// .git não deve aparecer em ByExt
	for k := range stats.ByExt {
		if strings.Contains(k, "git") {
			t.Errorf("extensão inesperada encontrada: %s", k)
		}
	}
}

func TestScan_BinaryNotCountedInLines(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir)

	v, ok := stats.ByExt[".bin"]
	if !ok {
		t.Error(".bin deve aparecer em ByExt mesmo sendo binário")
	}
	if v != 0 {
		t.Errorf("binário não deve ter linhas, obtido %d", v)
	}
}

func TestScan_ScanningFalseOnSuccess(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir)

	if stats.Scanning {
		t.Error("Scanning deve ser false para varredura concluída")
	}
}
