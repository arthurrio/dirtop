// chart.go
package main

import (
	"fmt"
	"strings"

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
	if len(values) == 0 || maxInt(values) == 0 {
		return blankBlock(width, height)
	}

	// Truncar se necessário: cada célula braille usa 2 valores
	maxValues := width * 2
	if len(values) > maxValues {
		values = values[len(values)-maxValues:]
	}

	maxVal := maxInt(values)
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
			bit := dotRow % 4
			if row >= 0 && row < chartRows {
				grid[row][colIdx] |= 1 << brailleLeft[bit]
			}
		}
		for dotRow := 0; dotRow < rightVal; dotRow++ {
			row := chartRows - 1 - dotRow/4
			bit := dotRow % 4
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
	right := "agora"
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
