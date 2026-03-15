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

// DefaultIntervals é a lista padrão de intervalos de atualização.
var DefaultIntervals = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

// metricDef descreve uma métrica exibível no gráfico.
type metricDef struct {
	name  string
	color string
}

// chartMetrics lista as métricas disponíveis para o gráfico (single-metric).
var chartMetrics = [3]metricDef{
	{"lines", ColorPurple},
	{"files", ColorBlue},
	{"dirs", ColorGreen},
}

// Model é o estado da aplicação bubbletea.
type Model struct {
	history     []Stats
	current     Stats
	cwd         string
	width       int
	height      int
	chartMode   ChartMode
	intervals   []time.Duration
	intervalIdx int
	metricIdx   int // index in chartMetrics; ignored in multi mode
}

// interval retorna o intervalo de atualização atual, com fallback seguro.
func (m Model) interval() time.Duration {
	if len(m.intervals) == 0 {
		return time.Second
	}
	return m.intervals[m.intervalIdx]
}

// metricValues extrai do histórico os valores da métrica atualmente selecionada.
func (m Model) metricValues() []int {
	vals := make([]int, len(m.history))
	for i, s := range m.history {
		switch m.metricIdx {
		case 1:
			vals[i] = s.Files
		case 2:
			vals[i] = s.Dirs
		default:
			vals[i] = s.Lines
		}
	}
	return vals
}

// formatInterval formata uma duração como "1s", "30s", "1m", "5m".
func formatInterval(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm", s/60)
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

// tickCmd agenda um tick após a duração especificada.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
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
		return m, tickCmd(m.interval())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "c":
			m.chartMode = (m.chartMode + 1) % chartModeCount
		case "i":
			if len(m.intervals) > 0 {
				m.intervalIdx = (m.intervalIdx + 1) % len(m.intervals)
			}
		case "m":
			if m.chartMode != ChartMultiLine {
				m.metricIdx = (m.metricIdx + 1) % len(chartMetrics)
			}
		}
	}

	return m, nil
}

// View renderiza o dashboard completo.
func (m Model) View() string {
	if m.width < 40 || m.height < 10 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			StyleGray.Render("Terminal too small"),
		)
	}

	var sb strings.Builder

	// --- Linha de status ---
	indicator := StyleGray.Render(fmt.Sprintf("↺ %s", formatInterval(m.interval())))
	if m.current.Scanning {
		indicator = StyleOrange.Render("↺ scanning...")
	}
	statusLine := fmt.Sprintf(" ▶ dirtop  [%s]  %s",
		StyleGray.Render(m.cwd),
		indicator,
	)
	sb.WriteString(statusLine)
	sb.WriteString("\n")

	// --- Métricas inline ---
	// A métrica ativa no gráfico (single-metric) é destacada com sua própria cor no label.
	active := m.metricIdx
	if m.chartMode == ChartMultiLine {
		active = -1 // nenhuma destacada individualmente no modo multi
	}
	labelFiles := StyleGray.Render("files")
	labelDirs := StyleGray.Render("│  dirs")
	labelLines := StyleGray.Render("│  lines")
	if active == 1 {
		labelFiles = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBlue)).Bold(true).Render("files")
	} else if active == 2 {
		labelDirs = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen)).Bold(true).Render("│  dirs")
	} else if active == 0 {
		labelLines = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple)).Bold(true).Render("│  lines")
	}
	metricsLine := fmt.Sprintf("  %s  %s    %s  %s    %s  %s",
		labelFiles,
		StyleBlue.Render(formatNumber(m.current.Files)),
		labelDirs,
		StyleGreen.Render(formatNumber(m.current.Dirs)),
		labelLines,
		StylePurple.Render(formatNumber(m.current.Lines)),
	)
	sb.WriteString(metricsLine)
	sb.WriteString("\n")

	// --- Separador ---
	sb.WriteString(StyleGray.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")

	// --- Histórico / Gráfico ---
	metricHint := fmt.Sprintf("  [m: %s]", chartMetrics[m.metricIdx].name)
	if m.chartMode == ChartMultiLine {
		metricHint = "" // multi mode shows all three, hint does not apply
	}
	histLabel := fmt.Sprintf(" HISTORY  [c: %s]%s", m.chartMode.Name(), metricHint)
	sb.WriteString(StyleGray.Render(histLabel))
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

	metricVals := m.metricValues()

	var chartStr string
	switch m.chartMode {
	case ChartSparkline:
		chartStr = RenderSparkline(metricVals, m.width, chartHeight)
	case ChartMultiLine:
		filesVals := make([]int, len(m.history))
		dirsVals := make([]int, len(m.history))
		linesVals := make([]int, len(m.history))
		for i, s := range m.history {
			filesVals[i] = s.Files
			dirsVals[i] = s.Dirs
			linesVals[i] = s.Lines
		}
		chartStr = RenderMultiLine(filesVals, dirsVals, linesVals, m.width, chartHeight)
	case ChartDelta:
		chartStr = RenderDelta(metricVals, m.width, chartHeight)
	case ChartHorizBar:
		chartStr = RenderHorizBar(metricVals, m.width, chartHeight, m.interval())
	default:
		chartStr = Render(metricVals, m.width, chartHeight)
	}
	sb.WriteString(chartStr)
	sb.WriteString("\n")

	// --- Separador ---
	sb.WriteString(StyleGray.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")

	// --- Extensões ---
	sb.WriteString(StyleGray.Render(" EXTENSIONS"))
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
		if k == "(no ext)" {
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
