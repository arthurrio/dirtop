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
	stats := Scan(dir, ScanOptions{})

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
	stats := Scan(dir, ScanOptions{})

	// main.go=3, README.md=3, Makefile=1, pkg/util.go=1, binary.bin=0 → 8 linhas
	if stats.Lines != 8 {
		t.Errorf("esperado 8 linhas, obtido %d", stats.Lines)
	}
}

func TestScan_GroupsByExtension(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir, ScanOptions{})

	// .go: main.go(3) + pkg/util.go(1) = 4
	if stats.ByExt[".go"] != 4 {
		t.Errorf("esperado .go=4, obtido %d", stats.ByExt[".go"])
	}

	// .md: README.md(3)
	if stats.ByExt[".md"] != 3 {
		t.Errorf("esperado .md=3, obtido %d", stats.ByExt[".md"])
	}

	// sem ext: Makefile(1)
	if stats.ByExt["(no ext)"] != 1 {
		t.Errorf("esperado (sem ext)=1, obtido %d", stats.ByExt["(no ext)"])
	}
}

func TestScan_IgnoresHiddenEntries(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir, ScanOptions{})

	// .git não deve aparecer em ByExt
	for k := range stats.ByExt {
		if strings.Contains(k, "git") {
			t.Errorf("extensão inesperada encontrada: %s", k)
		}
	}
}

func TestScan_BinaryNotCountedInLines(t *testing.T) {
	dir := createTestDir(t)
	stats := Scan(dir, ScanOptions{})

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
	stats := Scan(dir, ScanOptions{})

	if stats.Scanning {
		t.Error("Scanning deve ser false para varredura concluída")
	}
}

func TestScan_DevModeIgnoresCommonDependencyAndIDEDirs(t *testing.T) {
	dir := t.TempDir()

	nodeModules := filepath.Join(dir, "node_modules")
	if err := os.Mkdir(nodeModules, 0755); err != nil {
		t.Fatalf("erro ao criar node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nodeModules, "index.js"), []byte("console.log('x')\n"), 0644); err != nil {
		t.Fatalf("erro ao criar arquivo em node_modules: %v", err)
	}

	ideaDir := filepath.Join(dir, ".idea")
	if err := os.Mkdir(ideaDir, 0755); err != nil {
		t.Fatalf("erro ao criar .idea: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ideaDir, "workspace.xml"), []byte("<project />\n"), 0644); err != nil {
		t.Fatalf("erro ao criar arquivo em .idea: %v", err)
	}

	srcDir := filepath.Join(dir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("erro ao criar src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.ts"), []byte("const answer = 42\n"), 0644); err != nil {
		t.Fatalf("erro ao criar arquivo em src: %v", err)
	}

	stats := Scan(dir, ScanOptions{DevMode: true})

	if stats.Files != 1 {
		t.Errorf("esperado 1 arquivo com --dev, obtido %d", stats.Files)
	}
	if stats.Dirs != 1 {
		t.Errorf("esperado 1 diretório com --dev, obtido %d", stats.Dirs)
	}
	if stats.Lines != 1 {
		t.Errorf("esperado 1 linha com --dev, obtido %d", stats.Lines)
	}
	if got := stats.ByExt[".ts"]; got != 1 {
		t.Errorf("esperado .ts=1 com --dev, obtido %d", got)
	}
	if _, ok := stats.ByExt[".js"]; ok {
		t.Error(".js de node_modules não deve aparecer em ByExt com --dev")
	}
}

func TestScan_DevModeIgnoresGeneratedFilesAcrossEcosystems(t *testing.T) {
	dir := t.TempDir()

	srcDir := filepath.Join(dir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("erro ao criar src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("erro ao criar main.go: %v", err)
	}

	generatedFiles := []string{
		"App.iml",
		"Main.class",
		"native.so",
		"bundle.min.js.map",
	}
	for _, name := range generatedFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("generated\n"), 0644); err != nil {
			t.Fatalf("erro ao criar %s: %v", name, err)
		}
	}

	stats := Scan(dir, ScanOptions{DevMode: true})

	if stats.Files != 1 {
		t.Errorf("esperado 1 arquivo real com --dev, obtido %d", stats.Files)
	}
	if got := stats.ByExt[".go"]; got != 1 {
		t.Errorf("esperado .go=1 com --dev, obtido %d", got)
	}
	if _, ok := stats.ByExt[".iml"]; ok {
		t.Error(".iml não deve aparecer em ByExt com --dev")
	}
	if _, ok := stats.ByExt[".class"]; ok {
		t.Error(".class não deve aparecer em ByExt com --dev")
	}
	if _, ok := stats.ByExt[".so"]; ok {
		t.Error(".so não deve aparecer em ByExt com --dev")
	}
	if _, ok := stats.ByExt[".map"]; ok {
		t.Error("*.min.js.map não deve aparecer em ByExt com --dev")
	}
}

func TestScan_DevModeDoesNotIgnoreCommonDirsWhenDisabled(t *testing.T) {
	dir := t.TempDir()

	targetDir := filepath.Join(dir, "target")
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatalf("erro ao criar target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "app.jar"), []byte("jar-content\n"), 0644); err != nil {
		t.Fatalf("erro ao criar app.jar: %v", err)
	}

	stats := Scan(dir, ScanOptions{})

	if stats.Files != 1 {
		t.Errorf("esperado 1 arquivo sem --dev, obtido %d", stats.Files)
	}
	if stats.Dirs != 1 {
		t.Errorf("esperado 1 diretório sem --dev, obtido %d", stats.Dirs)
	}
	if got := stats.ByExt[".jar"]; got != 1 {
		t.Errorf("esperado .jar=1 sem --dev, obtido %d", got)
	}
}

func TestScan_DevModeDoesNotIgnoreProjectConfigFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"package.json":     "{\n  \"name\": \"app\"\n}\n",
		"pom.xml":          "<project></project>\n",
		"build.gradle.kts": "plugins {}\n",
		"tsconfig.json":    "{\n  \"compilerOptions\": {}\n}\n",
		"pyproject.toml":   "[project]\nname = \"app\"\n",
		"Cargo.toml":       "[package]\nname = \"app\"\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("erro ao criar %s: %v", name, err)
		}
	}

	stats := Scan(dir, ScanOptions{DevMode: true})

	if stats.Files != len(files) {
		t.Errorf("esperado %d arquivos de configuração com --dev, obtido %d", len(files), stats.Files)
	}
	if got := stats.ByExt[".json"]; got == 0 {
		t.Error(".json de configuração do projeto não deve ser ignorado com --dev")
	}
	if got := stats.ByExt[".xml"]; got == 0 {
		t.Error(".xml de configuração do projeto não deve ser ignorado com --dev")
	}
	if got := stats.ByExt[".toml"]; got == 0 {
		t.Error(".toml de configuração do projeto não deve ser ignorado com --dev")
	}
}
