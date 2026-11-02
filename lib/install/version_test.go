package install

import "testing"

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.26.38", "0.26.39", -1},
		{"0.26.39", "v0.26.39", 0},
		{"0.27.0", "0.26.39", 1},
		{"1.0", "0.99.99", 1},
		{"v1.2.3", "1.2.3", 0},
	}
	for _, tt := range tests {
		if got := compareVersion(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersion(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
