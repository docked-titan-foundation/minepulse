package btc

import (
	"math"
	"testing"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// ckpool renders hashrates through suffix_string(val, buf, 16, 0): "%.3g" plus a
// magnitude letter at or above 1000, a bare integer below it, and "%.3g" turning
// into exponent form at the top of a decade (research R2).
func TestParseSuffixed(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"0", 0},
		{"999", 999},
		{"1.5K", 1500},
		{"480G", 480e9},
		{"1.23P", 1.23e15},
		{"12.3T", 12.3e12},
		{"1e+03T", 1e15},
		{"3E", 3e18},
		{" 480G ", 480e9},
		{"480g", 480e9},
		{"", model.Unknown},
		{"not-a-number", model.Unknown},
		{"12X", model.Unknown},
		{"K", model.Unknown},
	}
	for _, tt := range tests {
		got := ParseSuffixed(tt.in)
		if tt.want == model.Unknown {
			if got != model.Unknown {
				t.Errorf("ParseSuffixed(%q) = %v, want Unknown", tt.in, got)
			}
			continue
		}
		if math.Abs(got-tt.want) > math.Abs(tt.want)*1e-9 {
			t.Errorf("ParseSuffixed(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// An unparseable hashrate must never read as an idle pool.
func TestParseSuffixedNeverZeroOnFailure(t *testing.T) {
	if got := ParseSuffixed("garbage"); got == 0 {
		t.Fatal("unparseable input returned 0, which renders as a stopped miner")
	}
}
