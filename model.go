// model.go
package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickMsg sinaliza que é hora de iniciar uma nova varredura.
type tickMsg time.Time

const historyMaxLen = 3600

// Model é o estado da aplicação bubbletea.
type Model struct {
	history []Stats
	current Stats
	cwd     string
	width   int
	height  int
}

// Init dispara a primeira varredura imediatamente, sem esperar 1 segundo.
func (m Model) Init() tea.Cmd {
	return scanCmd(m.cwd)
}

// scanCmd retorna um tea.Cmd que executa a varredura no path especificado.
func scanCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return ScanMsg(Scan(path))
	}
}

// tickCmd agenda um tick após 1 segundo.
func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update processa mensagens e atualiza o estado.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, scanCmd(m.cwd)

	case ScanMsg:
		stats := Stats(msg)
		if len(m.history) >= historyMaxLen {
			m.history = m.history[1:]
		}
		m.history = append(m.history, stats)
		m.current = stats
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renderiza o dashboard completo.
func (m Model) View() string {
	if m.width < 40 || m.height < 10 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			StyleGray.Render("Terminal muito pequeno"),
		)
	}

	var sb strings.Builder

	// --- Linha de status ---
	indicator := StyleGray.Render("↺ 1s")
	if m.current.Scanning {
		indicator = StyleOrange.Render("↺ scanning…")
	}
	statusLine := fmt.Sprintf(" ▶ monitor-cli  [%s]  %s",
		StyleGray.Render(m.cwd),
		indicator,
	)
	sb.WriteString(statusLine)
	sb.WriteString("\n")

	// --- Métricas inline ---
	metricsLine := fmt.Sprintf("  %s  %s    %s  %s    %s  %s",
		StyleGray.Render("arquivos"),
		StyleBlue.Render(formatNumber(m.current.Files)),
		StyleGray.Render("│  pastas"),
		StyleGreen.Render(formatNumber(m.current.Dirs)),
		StyleGray.Render("│  linhas"),
		StylePurple.Render(formatNumber(m.current.Lines)),
	)
	sb.WriteString(metricsLine)
	sb.WriteString("\n")

	// --- Separador ---
	sb.WriteString(StyleGray.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")

	// --- Histórico / Gráfico ---
	sb.WriteString(StyleGray.Render(" HISTÓRICO"))
	sb.WriteString("\n")

	// Calcular linhas de extensões (limitado ao espaço disponível)
	extCount := len(m.current.ByExt)
	extRows := (extCount + 1) / 2 // 2 colunas por linha
	if extRows < 1 {
		extRows = 1
	}

	// Overhead fixo: status(1) + metrics(1) + sep(1) + HISTÓRICO(1) + \n pós-chart(1) + sep(1) + EXTENSÕES(1) = 7
	const fixedOverhead = 7
	chartHeight := m.height - fixedOverhead - extRows
	if chartHeight < 5 {
		chartHeight = 5
	}

	values := make([]int, len(m.history))
	for i, s := range m.history {
		values[i] = s.Lines
	}
	sb.WriteString(Render(values, m.width, chartHeight))
	sb.WriteString("\n")

	// --- Separador ---
	sb.WriteString(StyleGray.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")

	// --- Extensões ---
	sb.WriteString(StyleGray.Render(" EXTENSÕES"))
	sb.WriteString("\n")
	sb.WriteString(renderExtensions(m.current.ByExt, m.width))

	return sb.String()
}

// extEntry representa uma entrada de extensão para ordenação.
type extEntry struct {
	name  string
	lines int
}

// renderExtensions renderiza o grid de extensões em 2 colunas.
func renderExtensions(byExt map[string]int, width int) string {
	if len(byExt) == 0 {
		return ""
	}

	var entries []extEntry
	var noExt *extEntry

	for k, v := range byExt {
		if k == "(sem ext)" {
			e := extEntry{k, v}
			noExt = &e
			continue
		}
		entries = append(entries, extEntry{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lines > entries[j].lines
	})
	if noExt != nil {
		entries = append(entries, *noExt)
	}

	leftColWidth := (width - 1) / 2
	rightColWidth := width - 1 - leftColWidth
	const countWidth = 8

	var sb strings.Builder
	for i := 0; i < len(entries); i += 2 {
		left := formatExtEntry(entries[i], leftColWidth, countWidth)
		right := ""
		if i+1 < len(entries) {
			right = formatExtEntry(entries[i+1], rightColWidth, countWidth)
		}
		sb.WriteString(" ")
		sb.WriteString(left)
		sb.WriteString(right)
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatExtEntry formata uma entrada de extensão para uma coluna.
func formatExtEntry(e extEntry, colWidth, countWidth int) string {
	nameWidth := colWidth - countWidth - 1

	name := e.name
	if utf8.RuneCountInString(name) > 12 {
		runes := []rune(name)
		name = string(runes[:11]) + "…"
	}

	nameStr := StyleBlue.Render(fmt.Sprintf("%-*s", nameWidth, name))
	countStr := fmt.Sprintf("%*s", countWidth, formatNumber(e.lines))

	return nameStr + countStr
}

// formatNumber formata um inteiro com separador de milhar pt-BR (".").
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
