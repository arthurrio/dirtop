// styles.go
package main

import "github.com/charmbracelet/lipgloss"

const (
	ColorBlue       = "#58a6ff" // valor arquivos, labels extensão
	ColorGreen      = "#3fb950" // valor pastas
	ColorPurple     = "#d2a8ff" // valor linhas totais, linha do gráfico
	ColorPurpleDark = "#3d1f5c" // área preenchida do gráfico
	ColorGray       = "#8b949e" // labels, eixos, indicador ↺ 1s
	ColorOrange     = "#e3b341" // indicador ↺ scanning…
)

var (
	StyleBlue   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBlue))
	StyleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
	StylePurple = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple))
	StyleGray   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	StyleOrange = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))
	StyleBold   = lipgloss.NewStyle().Bold(true)
	StyleDim    = lipgloss.NewStyle().Faint(true)
)
