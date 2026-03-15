// model_test.go
package main

import "testing"

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{142, "142"},
		{999, "999"},
		{1000, "1.000"},
		{8421, "8.421"},
		{100000, "100.000"},
		{1200000, "1.200.000"},
	}

	for _, c := range cases {
		result := formatNumber(c.input)
		if result != c.expected {
			t.Errorf("formatNumber(%d) = %q, esperado %q", c.input, result, c.expected)
		}
	}
}
