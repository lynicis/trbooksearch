package relevance

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestTitleScore(t *testing.T) {
	tests := []struct {
		title, query string
		want         float64
	}{
		// exact match
		{"Suç ve Ceza", "Suç ve Ceza", 1.0},
		// title contains all query words + extras
		{"Suç ve Ceza - Dostoyevski", "Suç ve Ceza", 1.0},
		// partial match (2 out of 3 query words)
		{"Suç ve Barış", "Suç ve Ceza", 0.67},
		// no match
		{"Kırmızı Pazartesi", "Suç ve Ceza", 0.0},
		// Turkish normalization: ş/ç handled
		{"Suc ve Ceza", "Suç ve Ceza", 1.0},
		// case-insensitive
		{"SUÇ VE CEZA", "suç ve ceza", 1.0},
		// empty query
		{"Suç ve Ceza", "", 0.0},
		// empty title
		{"", "Suç ve Ceza", 0.0},
	}
	for _, tt := range tests {
		got := TitleScore(tt.title, tt.query)
		if !almostEqual(got, tt.want) {
			t.Errorf("TitleScore(%q, %q) = %.2f, want %.2f", tt.title, tt.query, got, tt.want)
		}
	}
}

func TestFieldMatchScore(t *testing.T) {
	tests := []struct {
		field, query string
		want         float64
	}{
		{"Dostoyevski", "Dostoyevski", 1.0},
		{"Fyodor Dostoyevski", "Dostoyevski", 1.0},
		{"İş Bankası Kültür Yayınları", "İş Bankası", 1.0},
		{"Can Yayınları", "İş Bankası", 0.0},
		{"", "Dostoyevski", 0.0},
		{"Dostoyevski", "", 0.0},
	}
	for _, tt := range tests {
		got := FieldMatchScore(tt.field, tt.query)
		if !almostEqual(got, tt.want) {
			t.Errorf("FieldMatchScore(%q, %q) = %.2f, want %.2f", tt.field, tt.query, got, tt.want)
		}
	}
}

func TestComputeRelevance(t *testing.T) {
	// With only title query
	score := ComputeRelevance("Suç ve Ceza - Dostoyevski", "", "", "Suç ve Ceza", "", "")
	if score < 0.8 {
		t.Errorf("expected high relevance for matching title, got %.2f", score)
	}

	// With title + author match
	score2 := ComputeRelevance("Suç ve Ceza", "Dostoyevski", "", "Suç ve Ceza", "Dostoyevski", "")
	if score2 < score {
		t.Errorf("expected author match to boost score: title-only=%.2f, with-author=%.2f", score, score2)
	}

	// Irrelevant result
	score3 := ComputeRelevance("Python Programlama", "Guido", "Kodlab", "Suç ve Ceza", "", "")
	if score3 > 0.2 {
		t.Errorf("expected low relevance for irrelevant title, got %.2f", score3)
	}
}
