package mixer

import (
	"reflect"
	"sort"
	"testing"
)

func TestToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"apple", "Apple"},
		{"banana", "Banana"},
		{"PQS", "PQS"},
		{"pqs", "Pqs"},
		{"", ""},
		{"café", "Café"},
		{"niño", "Niño"},
		{"Über", "Über"},
	}
	for _, tt := range tests {
		got := toPascal(tt.input)
		if got != tt.want {
			t.Errorf("toPascal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToCamelFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Apple", "apple"},
		{"BANANA", "bANANA"},
		{"PQS", "pQS"},
		{"", ""},
		{"Café", "café"},
	}
	for _, tt := range tests {
		got := toCamelFirst(tt.input)
		if got != tt.want {
			t.Errorf("toCamelFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestApplyCasing(t *testing.T) {
	words := []string{"apple", "banana", "PQS"}

	pascal := ApplyCasing(words, 0)
	if !reflect.DeepEqual(pascal, []string{"Apple", "Banana", "PQS"}) {
		t.Errorf("PascalCase: got %v", pascal)
	}

	camel := ApplyCasing(words, 1)
	if !reflect.DeepEqual(camel, []string{"apple", "Banana", "PQS"}) {
		t.Errorf("camelCase: got %v", camel)
	}

	upper := ApplyCasing(words, 2)
	if !reflect.DeepEqual(upper, []string{"APPLE", "BANANA", "PQS"}) {
		t.Errorf("UPPER: got %v", upper)
	}

	lower := ApplyCasing(words, 3)
	if !reflect.DeepEqual(lower, []string{"apple", "banana", "pqs"}) {
		t.Errorf("lower: got %v", lower)
	}
}

func TestGeneratePermutations(t *testing.T) {
	tests := []struct {
		input []string
		want  int
	}{
		{[]string{"a"}, 1},
		{[]string{"a", "b"}, 2},
		{[]string{"a", "b", "c"}, 6},
		{[]string{"a", "b", "c", "d"}, 24},
		{[]string{}, 0},
		{nil, 0},
	}
	for _, tt := range tests {
		got := GeneratePermutations(tt.input)
		if len(got) != tt.want {
			t.Errorf("GeneratePermutations(%v) = %d perms, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestGeneratePermutationsUniqueness(t *testing.T) {
	perms := GeneratePermutations([]string{"a", "b", "c", "d"})
	seen := make(map[string]bool)
	for _, p := range perms {
		key := p[0] + p[1] + p[2] + p[3]
		if seen[key] {
			t.Errorf("Duplicate permutation: %s", key)
		}
		seen[key] = true
	}
}

func TestSubsets(t *testing.T) {
	tests := []struct {
		n, k    int
		wantLen int
	}{
		{3, 1, 3},
		{3, 2, 3},
		{3, 3, 1},
		{4, 2, 6},
		{5, 3, 10},
		{3, 4, 0},
		{3, 0, 0},
	}
	for _, tt := range tests {
		got := Subsets(tt.n, tt.k)
		if len(got) != tt.wantLen {
			t.Errorf("Subsets(%d, %d) = %d, want %d", tt.n, tt.k, len(got), tt.wantLen)
		}
	}
}

func TestSubsetsNoAliasing(t *testing.T) {
	subs := Subsets(5, 3)
	for _, s := range subs {
		if len(s) != 3 {
			t.Errorf("Subset has wrong length: %v", s)
		}
	}
}

func TestAllCombos(t *testing.T) {
	words := []string{"a", "b", "c"}
	combos := AllCombos(words, []int{2})
	if len(combos) != 6 {
		t.Errorf("AllCombos with size 2: got %d, want 6", len(combos))
	}

	combos = AllCombos(words, []int{2, 3})
	if len(combos) != 6+6 {
		t.Errorf("AllCombos with sizes 2,3: got %d, want 12", len(combos))
	}

	combos = AllCombos(words, []int{5})
	if len(combos) != 0 {
		t.Errorf("AllCombos with size > n: got %d, want 0", len(combos))
	}
}

func TestSpecialSubsets(t *testing.T) {
	subs := SpecialSubsets("ab", 2)
	if len(subs) != 4 {
		t.Errorf("SpecialSubsets('ab', 2): got %d, want 4 (a, b, ab, ba)", len(subs))
	}

	subs = SpecialSubsets("ab", 1)
	if len(subs) != 2 {
		t.Errorf("SpecialSubsets('ab', 1): got %d, want 2", len(subs))
	}
}

func TestParseNumberRanges(t *testing.T) {
	tests := []struct {
		input   string
		want    []int
		wantErr bool
	}{
		{"1-5", []int{1, 2, 3, 4, 5}, false},
		{"1,2,3", []int{1, 2, 3}, false},
		{"5-1", []int{1, 2, 3, 4, 5}, false},
		{"1-3 10", []int{1, 2, 3, 10}, false},
		{"1-3,1900-1902", []int{1, 2, 3, 1900, 1901, 1902}, false},
		{"69", []int{69}, false},
		{"", nil, false},
		{"abc", nil, true},
		{"1-abc", nil, true},
	}
	for _, tt := range tests {
		got, err := ParseNumberRanges(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseNumberRanges(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseNumberRanges(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDedupWords(t *testing.T) {
	raw := []string{"apple", "banana", "apple", "  cherry  ", "banana", ""}
	got := DedupWords(raw)
	want := []string{"apple", "banana", "cherry"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DedupWords(%v) = %v, want %v", raw, got, want)
	}
}

func TestDedupWordsBOM(t *testing.T) {
	raw := []string{"\ufeffapple", "banana"}
	got := DedupWords(raw)
	if len(got) != 2 || got[0] != "apple" {
		t.Errorf("DedupWords with BOM: got %v, want [apple banana]", got)
	}
}

func TestEstimateOutputs(t *testing.T) {
	cfg := &Config{
		Combos:     make([][]string, 10),
		CasingIdxs: []int{0, 1},
		AppendSpec: true,
		SpecCombos: make([]string, 5),
		AppendNums: true,
		Numbers:    make([]int, 3),
	}
	want := uint64(10 * 2 * 6 * 4)
	got := cfg.EstimateOutputs()
	if got != want {
		t.Errorf("EstimateOutputs() = %d, want %d", got, want)
	}
}

func TestAllCombosSortedProperty(t *testing.T) {
	words := []string{"a", "b"}
	combos := AllCombos(words, []int{2})
	results := make([]string, len(combos))
	for i, c := range combos {
		results[i] = c[0] + c[1]
	}
	sort.Strings(results)
	if results[0] != "ab" || results[1] != "ba" {
		t.Errorf("Expected [ab ba], got %v", results)
	}
}