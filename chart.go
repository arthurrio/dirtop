// chart.go
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Mapping of braille dots to bits in the U+2800 block.
// Left column: dots 1,2,3,7 -> bits 0,1,2,6
// Right column: dots 4,5,6,8 -> bits 3,4,5,7
// Each cell covers 2 columns x 4 rows.
var brailleLeft = [4]uint8{0, 1, 2, 6}  // Bits for rows 0-3 in the left column.
var brailleRight = [4]uint8{3, 4, 5, 7} // Bits for rows 0-3 in the right column.

// Render draws a line chart with a filled area using braille characters.
//
// The height parameter includes the X-axis row.
// The drawing area uses height-1 rows; the last row is always the X-axis.
//
// If values is empty or all zeros, it returns a width x height block of spaces.
func Render(values []int, width, height int) string {
	if height < 2 {
		height = 2
	}

	chartRows := height - 1 // Available rows for the braille chart.

	// Empty or all-zero input.
	maxVal := maxInt(values)
	if maxVal == 0 {
		return blankBlock(width, height)
	}

	// Truncate if needed: each braille cell uses 2 values.
	maxValues := width * 2
	if len(values) > maxValues {
		values = values[len(values)-maxValues:]
	}
	totalDots := chartRows * 4 // Available vertical dots.

	// Normalize values into [0, totalDots].
	normalized := make([]int, len(values))
	for i, v := range values {
		normalized[i] = v * totalDots / maxVal
	}

	// Build the braille cell grid: [row][column].
	cols := (len(normalized) + 1) / 2
	if cols > width {
		cols = width
	}

	// grid[row][col] = bitmask for the braille character.
	grid := make([][]uint8, chartRows)
	for i := range grid {
		grid[i] = make([]uint8, cols)
	}

	// Fill the grid.
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

		// Light dots from the baseline up to the value.
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

	// Determine the top row of each column for the line overlay.
	topRow := make([]int, cols)
	for colIdx := 0; colIdx < cols; colIdx++ {
		topRow[colIdx] = chartRows // Default: no value.
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

	// X-axis.
	sb.WriteString("\n")
	sb.WriteString(renderXAxis(width))

	return sb.String()
}

// renderXAxis returns the X-axis row with "t=0s" on the left and "now" on the right.
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

// blankBlock returns a block of spaces with exactly width x height characters.
func blankBlock(width, height int) string {
	row := strings.Repeat(" ", width)
	rows := make([]string, height)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// maxInt returns the largest value in an integer slice.
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

// ─── Chart Modes ─────────────────────────────────────────────────────────────

// ChartMode defines the history visualization type.
type ChartMode int

const (
	ChartHorizBar  ChartMode = iota // Horizontal block histogram (default).
	ChartBraille                    // Filled area.
	ChartSparkline                  // Line only, no fill.
	ChartMultiLine                  // Three overlaid metrics.
	ChartDelta                      // Delta versus the previous point.
	chartModeCount
)

// Name returns the user-facing chart mode name.
func (m ChartMode) Name() string {
	names := [...]string{"bars", "area", "line", "multi", "delta"}
	if int(m) < len(names) {
		return names[m]
	}
	return "?"
}

// ─── Sparkline ───────────────────────────────────────────────────────────────

// RenderSparkline renders only the chart line, without fill.
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

	// Light only the highest dot of each sub-column, without fill.
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

// ─── Multi-Line ──────────────────────────────────────────────────────────────

// RenderMultiLine renders three overlaid metrics in the same braille chart.
// Each series is normalized independently for better visibility.
// Color priority on cell conflicts: lines > files > dirs.
func RenderMultiLine(files, dirs, lines []int, width, height int) string {
	if height < 2 {
		height = 2
	}
	chartRows := height - 1
	maxValues := width * 2

	// Draw order: dirs (lowest priority) -> files -> lines (highest priority).
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
	// The legend replaces the X-axis.
	legendL := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple)).Render("─ lines")
	legendF := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBlue)).Render("─ files")
	legendD := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen)).Render("─ dirs")
	sb.WriteString(fmt.Sprintf(" %s   %s   %s", legendL, legendF, legendD))
	return sb.String()
}

// ─── Delta ───────────────────────────────────────────────────────────────────

// RenderDelta renders the delta between consecutive samples.
// Growth appears above the center line in green, decline below it in orange.
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

// ─── Horizontal Histogram ────────────────────────────────────────────────────

// formatAgo formats an offset duration as "-5s", "-2m", or "-1h".
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

// RenderHorizBar renders a horizontal histogram using block bars.
// Each row represents a change snapshot, with the most recent at the top.
// timestamps holds the real time each snapshot was taken; now is the reference.
func RenderHorizBar(values []int, timestamps []time.Time, width, height int, now time.Time) string {
	if height < 1 {
		height = 1
	}
	maxVal := maxInt(values)
	if maxVal == 0 {
		return blankBlock(width, height)
	}

	// Show the last `height` samples, most recent first.
	start := 0
	if len(values) > height {
		start = len(values) - height
	}
	samples := values[start:]
	times := timestamps[start:]
	n := len(samples)

	// Compute labelWidth from the oldest visible timestamp.
	var maxOffset time.Duration
	if n > 0 {
		maxOffset = now.Sub(times[0])
	}
	labelWidth := len(formatAgo(maxOffset))
	if labelWidth < len("now") {
		labelWidth = len("now")
	}

	const deltaWidth = 12 // Space for delta indicator like " +1.234".
	const valueWidth = 10 // Right-aligned formatted number.
	// Per-row layout: label + " " + bar + " " + value + delta.
	barArea := width - labelWidth - 2 - valueWidth - deltaWidth
	if barArea < 1 {
		barArea = 1
	}

	styleLabel := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	styleBar := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	styleValue := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	stylePlus := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
	styleMinus := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))

	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		sampleIdx := n - 1 - i // Most recent at the top.
		if sampleIdx < 0 {
			sb.WriteString(strings.Repeat(" ", width))
			continue
		}

		val := samples[sampleIdx]
		offset := now.Sub(times[sampleIdx])
		label := fmt.Sprintf("%*s", labelWidth, formatAgo(offset))

		barLen := val * barArea / maxVal
		bar := strings.Repeat("█", barLen) + strings.Repeat(" ", barArea-barLen)
		valueStr := fmt.Sprintf("%*s", valueWidth, formatNumber(val))

		// Delta indicator relative to the previous snapshot.
		deltaStr := strings.Repeat(" ", deltaWidth)
		if sampleIdx > 0 {
			diff := val - samples[sampleIdx-1]
			if diff > 0 {
				deltaStr = fmt.Sprintf("%*s", deltaWidth, "+"+formatNumber(diff))
				deltaStr = stylePlus.Render(deltaStr)
			} else if diff < 0 {
				deltaStr = fmt.Sprintf("%*s", deltaWidth, formatNumber(diff))
				deltaStr = styleMinus.Render(deltaStr)
			}
		}

		sb.WriteString(styleLabel.Render(label))
		sb.WriteString(" ")
		sb.WriteString(styleBar.Render(bar))
		sb.WriteString(" ")
		sb.WriteString(styleValue.Render(valueStr))
		sb.WriteString(deltaStr)
	}
	return sb.String()
}
