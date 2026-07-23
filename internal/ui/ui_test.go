package ui

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		want   string
	}{
		{"hello", 10, "hello     "},
		{"hello", 5, "hello"},
		{"hello", 3, "hello"},
		{"", 3, "   "},
	}
	for _, tt := range tests {
		got := PadRight(tt.input, tt.width)
		if got != tt.want {
			t.Errorf("PadRight(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
		}
	}
}

func TestColorize(t *testing.T) {
	SetNoColor(true)
	got := Colorize(Red, "hello")
	if got != "hello" {
		t.Errorf("Colorize with noColor: got %q, want %q", got, "hello")
	}
	SetNoColor(false)
	got = Colorize(Red, "hello")
	if got != Red+"hello"+Reset {
		t.Errorf("Colorize with color: got %q", got)
	}
}

func TestValidateOutputPath(t *testing.T) {
	err := ValidateOutputPath("output")
	if err != nil {
		t.Errorf("ValidateOutputPath('output') should be valid: %v", err)
	}

	err = ValidateOutputPath("../../etc/passwd")
	if err == nil {
		t.Error("ValidateOutputPath with '..' should fail")
	}
}

func TestValidateOutputPathTraversal(t *testing.T) {
	paths := []string{
		"../secret",
		"foo/../../bar",
		"./../escape",
	}
	for _, p := range paths {
		err := ValidateOutputPath(p)
		if err == nil {
			t.Errorf("ValidateOutputPath(%q) should reject traversal", p)
		}
	}
}