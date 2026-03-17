// main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	flagVersion := flag.Bool("version", false, "print version and exit")
	flagDev := flag.Bool("dev", false, "ignore common dependency, build, IDE, and cache directories")
	flagInterval := flag.Int("i", 0, "starting refresh interval in seconds (e.g. -i 5)")
	flagIntervals := flag.String("intervals", "", "available intervals in seconds, comma-separated (e.g. --intervals 1,5,10,30,60)")
	flag.Parse()

	if *flagVersion {
		fmt.Println("dirtop", version)
		os.Exit(0)
	}

	// Verificar acesso ao diretório atual antes de iniciar a TUI
	if _, err := os.ReadDir("."); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot access current directory: %v\n", err)
		os.Exit(1)
	}

	// Resolver path absoluto para exibição
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	intervals := parseIntervals(*flagIntervals, *flagInterval)
	intervalIdx := findIntervalIdx(intervals, *flagInterval)

	m := Model{
		cwd:         cwd,
		scanOpts:    ScanOptions{DevMode: *flagDev},
		width:       80,
		height:      24,
		intervals:   intervals,
		intervalIdx: intervalIdx,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// parseIntervals constrói a lista de intervalos a partir das flags.
// Se --intervals for fornecido, usa essa lista; caso contrário usa DefaultIntervals.
// Se -i for fornecido e não estiver na lista, insere-o na posição correta.
func parseIntervals(rawList string, startSecs int) []time.Duration {
	var durations []time.Duration

	if rawList != "" {
		for _, part := range strings.Split(rawList, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.Atoi(part)
			if err != nil || n <= 0 {
				fmt.Fprintf(os.Stderr, "warning: invalid interval ignored: %q\n", part)
				continue
			}
			durations = append(durations, time.Duration(n)*time.Second)
		}
	}

	if len(durations) == 0 {
		durations = make([]time.Duration, len(DefaultIntervals))
		copy(durations, DefaultIntervals)
	}

	// Garantir ordenação e unicidade
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	durations = dedupDurations(durations)

	// Inserir -i na lista caso não esteja presente
	if startSecs > 0 {
		target := time.Duration(startSecs) * time.Second
		found := false
		for _, d := range durations {
			if d == target {
				found = true
				break
			}
		}
		if !found {
			durations = append(durations, target)
			sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		}
	}

	return durations
}

// findIntervalIdx retorna o índice de startSecs na lista, ou 0 se não fornecido.
func findIntervalIdx(intervals []time.Duration, startSecs int) int {
	if startSecs <= 0 {
		return 0
	}
	target := time.Duration(startSecs) * time.Second
	for i, d := range intervals {
		if d == target {
			return i
		}
	}
	return 0
}

// dedupDurations remove durações duplicadas de uma slice já ordenada.
func dedupDurations(ds []time.Duration) []time.Duration {
	if len(ds) == 0 {
		return ds
	}
	out := ds[:1]
	for _, d := range ds[1:] {
		if d != out[len(out)-1] {
			out = append(out, d)
		}
	}
	return out
}
