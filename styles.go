// styles.go
package main

import "github.com/charmbracelet/lipgloss"

const (
	ColorBlue       = "#58a6ff" // File values and extension labels.
	ColorGreen      = "#3fb950" // Directory values.
	ColorPurple     = "#d2a8ff" // Total line counts and chart line.
	ColorPurpleDark = "#3d1f5c" // Filled chart area.
	ColorGray       = "#8b949e" // Labels, axes, and the ↺ 1s indicator.
	ColorOrange     = "#e3b341" // The ↺ scanning… indicator.
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
