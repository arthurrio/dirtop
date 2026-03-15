// chart.go
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Mapeamento dos pontos braille para bits no bloco U+2800.
// Coluna esquerda: pontos 1,2,3,7 → bits 0,1,2,6
// Coluna direita:  pontos 4,5,6,8 → bits 3,4,5,7
// Cada célula cobre 2 colunas × 4 linhas.
var brailleLeft = [4]uint8{0, 1, 2, 6}  // bits para linhas 0-3 da coluna esquerda
var brailleRight = [4]uint8{3, 4, 5, 7} // bits para linhas 0-3 da coluna direita

// Render renderiza um gráfico de linha com área preenchida usando caracteres braille.
//
// O parâmetro height inclui a linha do eixo X.
// A área de desenho usa height-1 linhas; a última linha é sempre o eixo X.
//
// Se values for vazio ou todos zeros, retorna um bloco de espaços width×height.
func Render(values []int, width, height int) string {
	if height < 2 {
		height = 2
	}

	chartRows := height - 1 // linhas disponíveis para o gráfico braille

	// Caso vazio ou todos zeros
	maxVal := maxInt(values)
	if maxVal == 0 {
		return blankBlock(width, height)
	}

	// Truncar se necessário: cada célula braille usa 2 valores
	maxValues := width * 2
	if len(values) > maxValues {
		values = values[len(values)-maxValues:]
	}
	totalDots := chartRows * 4 // pontos verticais disponíveis

	// Normalizar valores para [0, totalDots]
	normalized := make([]int, len(values))
	for i, v := range values {
		normalized[i] = v * totalDots / maxVal
	}

	// Construir grade de células braille: [linha][coluna]
	cols := (len(normalized) + 1) / 2
	if cols > width {
		cols = width
	}

	// grid[row][col] = bitmask do caractere braille
	grid := make([][]uint8, chartRows)
	for i := range grid {
		grid[i] = make([]uint8, cols)
	}

	// Preencher o grid
	for colIdx := 0; colIdx < cols; colIdx++ {
		leftIdx := colIdx * 2
		rightIdx := leftIdx + 1

		leftVal := 0
		if leftIdx < len(normalized) {
			leftVal = normalized[leftIdx]
		}
		rightVal := 0
		if rightIdx < len(normalized) {
			rightVal = normalized[rightIdx]
		}

		// Acender pontos da base até o valor
		for dotRow := 0; dotRow < leftVal; dotRow++ {
			row := chartRows - 1 - dotRow/4
			bit := 3 - (dotRow % 4)
			if row >= 0 && row < chartRows {
				grid[row][colIdx] |= 1 << brailleLeft[bit]
			}
		}
		for dotRow := 0; dotRow < rightVal; dotRow++ {
			row := chartRows - 1 - dotRow/4
			bit := 3 - (dotRow % 4)
			if row >= 0 && row < chartRows {
				grid[row][colIdx] |= 1 << brailleRight[bit]
			}
		}
	}

	// Determinar a linha "topo" de cada coluna (linha vs área)
	topRow := make([]int, cols)
	for colIdx := 0; colIdx < cols; colIdx++ {
		topRow[colIdx] = chartRows // default: sem valor
		for row := 0; row < chartRows; row++ {
			if grid[row][colIdx] != 0 {
				topRow[colIdx] = row
				break
			}
		}
	}

	styleLine := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	styleFill := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurpleDark))

	var sb strings.Builder

	for row := 0; row < chartRows; row++ {
		line := make([]string, width)
		for i := range line {
			line[i] = " "
		}

		for colIdx := 0; colIdx < cols; colIdx++ {
			mask := grid[row][colIdx]
			if mask == 0 {
				continue
			}
			ch := string(rune(0x2800 + int(mask)))
			if row == topRow[colIdx] {
				line[colIdx] = styleLine.Render(ch)
			} else {
				line[colIdx] = styleFill.Render(ch)
			}
		}

		if row > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Join(line, ""))
	}

	// Eixo X
	sb.WriteString("\n")
	sb.WriteString(renderXAxis(width))

	return sb.String()
}

// renderXAxis retorna a linha do eixo X com "t=0s" à esquerda e "agora" à direita.
func renderXAxis(width int) string {
	left := "t=0s"
	right := "now"
	sepLen := width - len(left) - len(right)
	if sepLen < 0 {
		sepLen = 0
	}
	sep := strings.Repeat("─", sepLen)
	styleAxis := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	return fmt.Sprintf("%s%s%s",
		styleAxis.Render(left),
		styleAxis.Render(sep),
		styleAxis.Render(right),
	)
}

// blankBlock retorna um bloco de espaços com exatamente width×height caracteres.
func blankBlock(width, height int) string {
	row := strings.Repeat(" ", width)
	rows := make([]string, height)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// maxInt retorna o maior valor de um slice de inteiros.
func maxInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// ─── Modos de gráfico ────────────────────────────────────────────────────────

// ChartMode define o tipo de visualização do histórico.
type ChartMode int

const (
	ChartBraille   ChartMode = iota // área preenchida (padrão)
	ChartSparkline                  // apenas a linha, sem preenchimento
	ChartMultiLine                  // três métricas sobrepostas
	ChartDelta                      // variação (Δ) em relação ao ponto anterior
	ChartHorizBar                   // histograma horizontal de blocos
	chartModeCount
)

// Name retorna o nome legível do modo de gráfico.
func (m ChartMode) Name() string {
	names := [...]string{"area", "line", "multi", "delta", "bars"}
	if int(m) < len(names) {
		return names[m]
	}
	return "?"
}

// ─── Sparkline ───────────────────────────────────────────────────────────────

// RenderSparkline renderiza apenas a linha do gráfico, sem preenchimento.
func RenderSparkline(values []int, width, height int) string {
	if height < 2 {
		height = 2
	}
	chartRows := height - 1
	maxVal := maxInt(values)
	if maxVal == 0 {
		return blankBlock(width, height)
	}
	maxValues := width * 2
	if len(values) > maxValues {
		values = values[len(values)-maxValues:]
	}
	totalDots := chartRows * 4
	normalized := make([]int, len(values))
	for i, v := range values {
		n := v * totalDots / maxVal
		if n == 0 && v > 0 {
			n = 1
		}
		normalized[i] = n
	}
	cols := (len(normalized) + 1) / 2
	if cols > width {
		cols = width
	}
	grid := make([][]uint8, chartRows)
	for i := range grid {
		grid[i] = make([]uint8, cols)
	}

	// Acende apenas o ponto mais alto de cada sub-coluna (sem preenchimento)
	setTopBit := func(colIdx, val int, bits [4]uint8) {
		if val <= 0 {
			return
		}
		dotRow := val - 1
		row := chartRows - 1 - dotRow/4
		bit := 3 - (dotRow % 4)
		if row >= 0 && row < chartRows {
			grid[row][colIdx] |= 1 << bits[bit]
		}
	}
	for colIdx := 0; colIdx < cols; colIdx++ {
		leftVal, rightVal := 0, 0
		if colIdx*2 < len(normalized) {
			leftVal = normalized[colIdx*2]
		}
		if colIdx*2+1 < len(normalized) {
			rightVal = normalized[colIdx*2+1]
		}
		setTopBit(colIdx, leftVal, brailleLeft)
		setTopBit(colIdx, rightVal, brailleRight)
	}

	styleLine := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	var sb strings.Builder
	for row := 0; row < chartRows; row++ {
		if row > 0 {
			sb.WriteString("\n")
		}
		line := make([]string, width)
		for i := range line {
			line[i] = " "
		}
		for colIdx := 0; colIdx < cols; colIdx++ {
			if mask := grid[row][colIdx]; mask != 0 {
				line[colIdx] = styleLine.Render(string(rune(0x2800 + int(mask))))
			}
		}
		sb.WriteString(strings.Join(line, ""))
	}
	sb.WriteString("\n")
	sb.WriteString(renderXAxis(width))
	return sb.String()
}

// ─── Multi-linha ─────────────────────────────────────────────────────────────

// RenderMultiLine renderiza três métricas sobrepostas no mesmo gráfico braille.
// Cada série é normalizada de forma independente para melhor visibilidade.
// Prioridade de cor em conflito de célula: linhas > arquivos > pastas.
func RenderMultiLine(files, dirs, lines []int, width, height int) string {
	if height < 2 {
		height = 2
	}
	chartRows := height - 1
	maxValues := width * 2

	// Ordem de desenho: dirs (menor prioridade) → files → lines (maior prioridade)
	type series struct {
		vals  []int
		color string
	}
	allSeries := []series{
		{dirs, ColorGreen},
		{files, ColorBlue},
		{lines, ColorPurple},
	}

	maskGrid := make([][]uint8, chartRows)
	colorGrid := make([][]string, chartRows)
	for i := range maskGrid {
		maskGrid[i] = make([]uint8, width)
		colorGrid[i] = make([]string, width)
	}

	for _, s := range allSeries {
		vals := s.vals
		maxVal := maxInt(vals)
		if maxVal == 0 {
			continue
		}
		if len(vals) > maxValues {
			vals = vals[len(vals)-maxValues:]
		}
		totalDots := chartRows * 4
		normalized := make([]int, len(vals))
		for i, v := range vals {
			n := v * totalDots / maxVal
			if n == 0 && v > 0 {
				n = 1
			}
			normalized[i] = n
		}
		cols := (len(normalized) + 1) / 2
		if cols > width {
			cols = width
		}
		setTopBit := func(colIdx, val int, bits [4]uint8) {
			if val <= 0 || colIdx >= width {
				return
			}
			dotRow := val - 1
			row := chartRows - 1 - dotRow/4
			bit := 3 - (dotRow % 4)
			if row >= 0 && row < chartRows {
				maskGrid[row][colIdx] |= 1 << bits[bit]
				colorGrid[row][colIdx] = s.color
			}
		}
		for colIdx := 0; colIdx < cols; colIdx++ {
			leftVal, rightVal := 0, 0
			if colIdx*2 < len(normalized) {
				leftVal = normalized[colIdx*2]
			}
			if colIdx*2+1 < len(normalized) {
				rightVal = normalized[colIdx*2+1]
			}
			setTopBit(colIdx, leftVal, brailleLeft)
			setTopBit(colIdx, rightVal, brailleRight)
		}
	}

	var sb strings.Builder
	for row := 0; row < chartRows; row++ {
		if row > 0 {
			sb.WriteString("\n")
		}
		line := make([]string, width)
		for i := range line {
			line[i] = " "
		}
		for colIdx := 0; colIdx < width; colIdx++ {
			mask := maskGrid[row][colIdx]
			if mask == 0 {
				continue
			}
			ch := string(rune(0x2800 + int(mask)))
			line[colIdx] = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrid[row][colIdx])).Render(ch)
		}
		sb.WriteString(strings.Join(line, ""))
	}
	sb.WriteString("\n")
	// Legenda substitui o eixo X
	legendL := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple)).Render("─ lines")
	legendF := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBlue)).Render("─ files")
	legendD := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen)).Render("─ dirs")
	sb.WriteString(fmt.Sprintf(" %s   %s   %s", legendL, legendF, legendD))
	return sb.String()
}

// ─── Delta ───────────────────────────────────────────────────────────────────

// RenderDelta renderiza a variação (Δ) entre amostras consecutivas.
// Crescimento aparece acima da linha central (verde), queda abaixo (laranja).
func RenderDelta(values []int, width, height int) string {
	if height < 2 {
		height = 2
	}
	if len(values) < 2 {
		return blankBlock(width, height)
	}
	chartRows := height - 1

	deltas := make([]int, len(values)-1)
	for i := 1; i < len(values); i++ {
		deltas[i-1] = values[i] - values[i-1]
	}
	maxAbs := 0
	for _, d := range deltas {
		if d < 0 {
			d = -d
		}
		if d > maxAbs {
			maxAbs = d
		}
	}
	if maxAbs == 0 {
		return blankBlock(width, height)
	}
	maxValues := width * 2
	if len(deltas) > maxValues {
		deltas = deltas[len(deltas)-maxValues:]
	}

	halfDots := (chartRows * 4) / 2
	normalized := make([]int, len(deltas))
	for i, d := range deltas {
		normalized[i] = d * halfDots / maxAbs
	}
	cols := (len(normalized) + 1) / 2
	if cols > width {
		cols = width
	}

	grid := make([][]uint8, chartRows)
	signGrid := make([][]int, chartRows) // 1 = positivo, -1 = negativo
	for i := range grid {
		grid[i] = make([]uint8, cols)
		signGrid[i] = make([]int, cols)
	}

	center := halfDots
	fillDots := func(colIdx, val int, bits [4]uint8, sign int) {
		var start, end int
		if val >= 0 {
			start = center
			end = center + val
		} else {
			start = center + val
			end = center
		}
		for dotRow := start; dotRow < end; dotRow++ {
			row := chartRows - 1 - dotRow/4
			bit := 3 - (dotRow % 4)
			if row >= 0 && row < chartRows {
				grid[row][colIdx] |= 1 << bits[bit]
				signGrid[row][colIdx] = sign
			}
		}
	}
	for colIdx := 0; colIdx < cols; colIdx++ {
		leftVal, rightVal := 0, 0
		if colIdx*2 < len(normalized) {
			leftVal = normalized[colIdx*2]
		}
		if colIdx*2+1 < len(normalized) {
			rightVal = normalized[colIdx*2+1]
		}
		leftSign, rightSign := 1, 1
		if leftVal < 0 {
			leftSign = -1
		}
		if rightVal < 0 {
			rightSign = -1
		}
		fillDots(colIdx, leftVal, brailleLeft, leftSign)
		fillDots(colIdx, rightVal, brailleRight, rightSign)
	}

	centerGridRow := chartRows - 1 - center/4
	stylePos := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
	styleNeg := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))
	styleCenter := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))

	var sb strings.Builder
	for row := 0; row < chartRows; row++ {
		if row > 0 {
			sb.WriteString("\n")
		}
		line := make([]string, width)
		for i := range line {
			if row == centerGridRow {
				line[i] = styleCenter.Render("·")
			} else {
				line[i] = " "
			}
		}
		for colIdx := 0; colIdx < cols; colIdx++ {
			mask := grid[row][colIdx]
			if mask == 0 {
				continue
			}
			ch := string(rune(0x2800 + int(mask)))
			if signGrid[row][colIdx] >= 0 {
				line[colIdx] = stylePos.Render(ch)
			} else {
				line[colIdx] = styleNeg.Render(ch)
			}
		}
		sb.WriteString(strings.Join(line, ""))
	}
	sb.WriteString("\n")
	sb.WriteString(renderXAxis(width))
	return sb.String()
}

// ─── Histograma horizontal ───────────────────────────────────────────────────

// formatAgo formata uma duração de offset como "-5s", "-2m", "-1h".
func formatAgo(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("-%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("-%dm", s/60)
	}
	return fmt.Sprintf("-%dh", s/3600)
}

// RenderHorizBar renderiza um histograma horizontal com barras de blocos.
// Cada linha representa uma amostra; a mais recente fica no topo.
// interval é o tempo real entre cada amostra, usado para calcular os labels de tempo.
func RenderHorizBar(values []int, width, height int, interval time.Duration) string {
	if interval <= 0 {
		interval = time.Second
	}
	if height < 1 {
		height = 1
	}
	maxVal := maxInt(values)
	if maxVal == 0 {
		return blankBlock(width, height)
	}

	// Mostrar os últimos `height` samples (mais recente no topo)
	samples := values
	if len(samples) > height {
		samples = samples[len(samples)-height:]
	}
	n := len(samples)

	// Calcular labelWidth dinamicamente pelo pior caso visível
	maxOffset := time.Duration(n-1) * interval
	labelWidth := len(formatAgo(maxOffset))
	if labelWidth < len("now") {
		labelWidth = len("now")
	}

	const valueWidth = 10 // número formatado alinhado à direita
	// layout por linha: label + " " + bar + " " + value
	barArea := width - labelWidth - 2 - valueWidth
	if barArea < 1 {
		barArea = 1
	}

	styleLabel := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	styleBar := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	styleValue := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))

	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		sampleIdx := n - 1 - i // índice 0 = mais recente no topo
		if sampleIdx < 0 {
			sb.WriteString(strings.Repeat(" ", width))
			continue
		}

		val := samples[sampleIdx]
		offset := time.Duration(n-1-sampleIdx) * interval
		label := fmt.Sprintf("%*s", labelWidth, formatAgo(offset))

		barLen := val * barArea / maxVal
		bar := strings.Repeat("█", barLen) + strings.Repeat(" ", barArea-barLen)
		valueStr := fmt.Sprintf("%*s", valueWidth, formatNumber(val))

		sb.WriteString(styleLabel.Render(label))
		sb.WriteString(" ")
		sb.WriteString(styleBar.Render(bar))
		sb.WriteString(" ")
		sb.WriteString(styleValue.Render(valueStr))
	}
	return sb.String()
}
