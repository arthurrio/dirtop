// model_test.go
package main

import (
	"testing"
	"time"
)

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

func TestStatsChanged(t *testing.T) {
	base := Stats{Files: 10, Dirs: 3, Lines: 100}

	cases := []struct {
		name    string
		a, b    Stats
		changed bool
	}{
		{"identical", base, base, false},
		{"files differ", base, Stats{Files: 11, Dirs: 3, Lines: 100}, true},
		{"dirs differ", base, Stats{Files: 10, Dirs: 4, Lines: 100}, true},
		{"lines differ", base, Stats{Files: 10, Dirs: 3, Lines: 101}, true},
		{"scanning flag ignored", base, Stats{Files: 10, Dirs: 3, Lines: 100, Scanning: true}, false},
	}

	for _, c := range cases {
		if got := statsChanged(c.a, c.b); got != c.changed {
			t.Errorf("%s: statsChanged = %v, expected %v", c.name, got, c.changed)
		}
	}
}

func TestUpdate_AppendsHistoryOnlyOnChange(t *testing.T) {
	m := Model{intervals: []time.Duration{time.Second}}

	// First scan should always be recorded.
	initial := Stats{Files: 5, Dirs: 2, Lines: 50, ByExt: map[string]int{}}
	updated, _ := m.Update(ScanMsg(initial))
	m = updated.(Model)
	if len(m.history) != 1 {
		t.Fatalf("expected 1 history entry after first scan, got %d", len(m.history))
	}

	// Same scan should not create a new history entry.
	updated, _ = m.Update(ScanMsg(initial))
	m = updated.(Model)
	if len(m.history) != 1 {
		t.Errorf("expected history to stay at 1 when stats unchanged, got %d", len(m.history))
	}

	// A changed scan should append a new entry.
	changed := Stats{Files: 6, Dirs: 2, Lines: 50, ByExt: map[string]int{}}
	updated, _ = m.Update(ScanMsg(changed))
	m = updated.(Model)
	if len(m.history) != 2 {
		t.Errorf("expected 2 history entries after a change, got %d", len(m.history))
	}

	// Current stats must always reflect the latest scan, even without a new entry.
	updated, _ = m.Update(ScanMsg(changed))
	m = updated.(Model)
	if m.current.Files != 6 {
		t.Errorf("expected current.Files=6, got %d", m.current.Files)
	}
	if len(m.history) != 2 {
		t.Errorf("expected history to stay at 2 when stats repeat, got %d", len(m.history))
	}
}

func TestUpdate_RecordsTimestampOnChange(t *testing.T) {
	m := Model{intervals: []time.Duration{time.Second}}
	before := time.Now()

	updated, _ := m.Update(ScanMsg(Stats{Files: 1, ByExt: map[string]int{}}))
	m = updated.(Model)

	after := time.Now()
	if len(m.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(m.history))
	}
	ts := m.history[0].Time
	if ts.Before(before) || ts.After(after) {
		t.Errorf("snapshot timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}
