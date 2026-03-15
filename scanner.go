// scanner.go
package main

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Stats contém as métricas coletadas de uma varredura.
type Stats struct {
	Files    int
	Dirs     int
	Lines    int
	ByExt    map[string]int // ".go" → linhas; "(sem ext)" para sem extensão
	Scanning bool           // true se varredura foi interrompida por timeout
}

// ScanMsg é enviada ao modelo bubbletea após uma varredura.
type ScanMsg Stats

// Scan percorre o diretório path e retorna as métricas coletadas.
// A varredura tem timeout de 5 segundos; se expirar, retorna dados parciais
// com Stats.Scanning = true.
func Scan(path string) Stats {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := Stats{
		ByExt: make(map[string]int),
	}

	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		// Verificar timeout
		select {
		case <-ctx.Done():
			stats.Scanning = true
			return filepath.SkipAll
		default:
		}

		// Silenciar todos os erros de I/O
		if err != nil {
			return nil
		}

		name := d.Name()

		// Ignorar entradas ocultas
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Ignorar symlinks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			// Não contar o diretório raiz
			if p != path {
				stats.Dirs++
			}
			return nil
		}

		// É um arquivo regular
		stats.Files++

		ext := filepath.Ext(name)
		if ext == "" {
			ext = "(sem ext)"
		}

		// Detectar se é texto
		if isTextFile(p) {
			lines := countLines(ctx, p)
			stats.Lines += lines
			stats.ByExt[ext] += lines
		} else {
			// Arquivo binário: conta no mapa com 0 linhas (garante chave presente)
			if _, ok := stats.ByExt[ext]; !ok {
				stats.ByExt[ext] = 0
			}
		}

		return nil
	})

	return stats
}

// isTextFile retorna true se o arquivo não contiver byte nulo nos primeiros 512 bytes.
func isTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		// Empty file — treat as text (0 lines)
		return true
	}

	for _, b := range buf[:n] {
		if b == 0x00 {
			return false
		}
	}
	return true
}

// countLines conta o número de linhas de um arquivo texto.
func countLines(ctx context.Context, path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
		// Check timeout every 1000 lines to avoid blocking indefinitely
		if count%1000 == 0 {
			select {
			case <-ctx.Done():
				return count
			default:
			}
		}
	}
	return count
}
