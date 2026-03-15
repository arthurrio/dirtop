// main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Verificar acesso ao diretório atual antes de iniciar a TUI
	if _, err := os.ReadDir("."); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: não foi possível acessar o diretório atual: %v\n", err)
		os.Exit(1)
	}

	// Resolver path absoluto para exibição
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	m := Model{
		cwd:    cwd,
		width:  80,
		height: 24,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar: %v\n", err)
		os.Exit(1)
	}
}
