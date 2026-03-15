// chart_test.go
package main

import (
	"strings"
	"testing"
)

func TestRender_EmptyInput(t *testing.T) {
	result := Render([]int{}, 20, 5)
	lines := strings.Split(result, "\n")

	// Deve retornar exatamente `height` linhas
	if len(lines) != 5 {
		t.Errorf("esperado 5 linhas, obtido %d", len(lines))
	}

	// Cada linha deve ter largura width (espaços)
	for i, line := range lines {
		stripped := stripANSI(line)
		if len([]rune(stripped)) != 20 {
			t.Errorf("linha %d: esperado 20 chars, obtido %d: %q", i, len([]rune(stripped)), stripped)
		}
	}
}

func TestRender_AllZeros(t *testing.T) {
	result := Render([]int{0, 0, 0, 0}, 20, 5)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Errorf("esperado 5 linhas, obtido %d", len(lines))
	}
}

func TestRender_XAxisLabels(t *testing.T) {
	result := Render([]int{1, 2, 3}, 30, 5)
	lines := strings.Split(result, "\n")
	lastLine := stripANSI(lines[len(lines)-1])

	if !strings.HasPrefix(lastLine, "t=0s") {
		t.Errorf("última linha deve começar com 't=0s', obtido: %q", lastLine)
	}
	if !strings.HasSuffix(strings.TrimRight(lastLine, " "), "agora") {
		t.Errorf("última linha deve terminar com 'agora', obtido: %q", lastLine)
	}
}

func TestRender_OutputHeightMatchesParam(t *testing.T) {
	for _, h := range []int{5, 8, 12, 20} {
		result := Render([]int{10, 20, 30, 40, 50}, 40, h)
		lines := strings.Split(result, "\n")
		if len(lines) != h {
			t.Errorf("height=%d: esperado %d linhas, obtido %d", h, h, len(lines))
		}
	}
}

func TestRender_TruncatesLongHistory(t *testing.T) {
	// width=10 → suporta 20 valores (2 por célula braille)
	// passar 30 valores não deve causar pânico
	values := make([]int, 30)
	for i := range values {
		values[i] = i * 10
	}
	result := Render(values, 10, 5)
	if result == "" {
		t.Error("resultado não deve ser vazio")
	}
}

// stripANSI remove sequências de escape ANSI de uma string.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
