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

// DefaultDevIgnoreDirs lists generated, external, or local-only directories
// that should be ignored when the CLI runs with --dev.
// Note: hidden directories (prefixed with ".") are already skipped by the
// scanner, so they do not need to appear here.
var DefaultDevIgnoreDirs = []string{
	"_build",
	"__pycache__",
	"CMakeFiles",
	"DerivedData",
	"Pods",
	"bin",
	"build",
	"coverage",
	"cmake-build-debug",
	"cmake-build-release",
	"deps",
	"dist",
	"log",
	"logs",
	"node_modules",
	"obj",
	"out",
	"site",
	"target",
	"temp",
	"tmp",
	"vendor",
	"venv",
}

// DefaultDevIgnoreExts lists generated file extensions that should be ignored with --dev.
var DefaultDevIgnoreExts = []string{
	".class",
	".dll",
	".dylib",
	".exe",
	".iml",
	".o",
	".obj",
	".pyd",
	".pyc",
	".pyo",
	".so",
	".test",
}

// ScanOptions controls additional scan rules.
type ScanOptions struct {
	DevMode    bool
	IgnoreDirs []string
}

// Stats contains the metrics collected from a scan.
type Stats struct {
	Files    int
	Dirs     int
	Lines    int
	ByExt    map[string]int // ".go" -> lines; "(no ext)" for files without an extension
	Scanning bool           // true if the scan was interrupted by timeout
}

// ScanMsg is sent to the Bubble Tea model after a scan completes.
type ScanMsg Stats

// Scan walks the given path and returns the collected metrics.
// The scan has a 5-second timeout; if it expires, it returns partial data
// with Stats.Scanning = true.
func Scan(path string, opts ScanOptions) Stats {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := Stats{
		ByExt: make(map[string]int),
	}

	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		// Check for timeout.
		select {
		case <-ctx.Done():
			stats.Scanning = true
			return filepath.SkipAll
		default:
		}

		// Silence all I/O errors.
		if err != nil {
			return nil
		}

		name := d.Name()

		// Ignore hidden entries.
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldIgnoreEntry(name, d.IsDir(), opts) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Ignore symlinks.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			// Do not count the root directory.
			if p != path {
				stats.Dirs++
			}
			return nil
		}

		// Regular file.
		stats.Files++

		ext := filepath.Ext(name)
		if ext == "" {
			ext = "(no ext)"
		}

		// Detect whether the file is text.
		if isTextFile(p) {
			lines := countLines(ctx, p)
			stats.Lines += lines
			stats.ByExt[ext] += lines
		} else {
			// Binary file: keep the extension in the map with 0 lines.
			if _, ok := stats.ByExt[ext]; !ok {
				stats.ByExt[ext] = 0
			}
		}

		return nil
	})

	return stats
}

func shouldIgnoreEntry(name string, isDir bool, opts ScanOptions) bool {
	if isDir {
		return shouldIgnoreDir(name, opts)
	}

	if !opts.DevMode {
		return false
	}

	ext := filepath.Ext(name)
	if ext != "" && containsName(DefaultDevIgnoreExts, ext) {
		return true
	}

	return strings.HasSuffix(name, ".min.js.map")
}

func shouldIgnoreDir(name string, opts ScanOptions) bool {
	if containsName(opts.IgnoreDirs, name) {
		return true
	}
	if !opts.DevMode {
		return false
	}
	return containsName(DefaultDevIgnoreDirs, name)
}

func containsName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

// isTextFile returns true if the file does not contain a NUL byte in the first 512 bytes.
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

// countLines counts the number of lines in a text file.
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
