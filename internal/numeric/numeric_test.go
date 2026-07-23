package numeric

import "testing"

func TestCountValid(t *testing.T) {
	tests := []struct {
		n    int
		want uint64
	}{
		{0, 0},
		{-1, 0},
		{1, 10},
		{2, 100},
		{3, 990},
		{4, 9810},
		{5, 97200},
		{6, 963090},
	}
	for _, tt := range tests {
		got := CountValid(tt.n)
		if got != tt.want {
			t.Errorf("CountValid(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestCountValidBruteForce(t *testing.T) {
	total := uint64(0)
	for a := 0; a < 10; a++ {
		for b := 0; b < 10; b++ {
			for c := 0; c < 10; c++ {
				for d := 0; d < 10; d++ {
					if !((a == b && b == c) || (b == c && c == d)) {
						total++
					}
				}
			}
		}
	}
	got := CountValid(4)
	if got != total {
		t.Errorf("CountValid(4) = %d, brute force = %d", got, total)
	}
}