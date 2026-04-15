package relevance

import "testing"

func TestNormalizeTurkish(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Suç ve Ceza", "suc ve ceza"},
		{"İSTANBUL", "istanbul"},
		{"IŞIK", "isik"},
		{"Güneş", "gunes"},
		{"Öğretmen", "ogretmen"},
		{"Üç", "uc"},
		{"Şeker", "seker"},
		{"çiçek", "cicek"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizeTurkish(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeTurkish(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"Suç ve Ceza", []string{"suc", "ve", "ceza"}},
		{"İstanbul'un Çocukları", []string{"istanbulun", "cocuklari"}},
		{"  ", nil},
		{"one", []string{"one"}},
	}
	for _, tt := range tests {
		got := Tokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("Tokenize(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
