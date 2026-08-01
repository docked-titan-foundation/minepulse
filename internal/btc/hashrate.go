package btc

import (
	"strconv"
	"strings"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// magnitudes are ckpool's suffix_string() letters, in H/s.
var magnitudes = map[byte]float64{
	'K': 1e3, 'M': 1e6, 'G': 1e9, 'T': 1e12, 'P': 1e15, 'E': 1e18,
}

// ParseSuffixed converts a ckpool hashrate string to H/s. ckpool writes these
// with suffix_string(val, buf, 16, 0), so the forms are "0", "999" (no suffix
// below 1000), "1.5K", "480G", and — because %.3g flips to exponent notation at
// the top of a decade — "1e+03T".
//
// Anything it cannot read returns model.Unknown, never 0: a zero hashrate is a
// claim that the pool is idle, and a parse failure is not evidence of that.
func ParseSuffixed(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return model.Unknown
	}
	mult := 1.0
	last := s[len(s)-1]
	if last >= 'a' && last <= 'z' {
		last -= 'a' - 'A'
	}
	if m, ok := magnitudes[last]; ok {
		mult = m
		s = strings.TrimSpace(s[:len(s)-1])
		if s == "" {
			return model.Unknown
		}
	} else if last < '0' || last > '9' {
		return model.Unknown // a suffix we do not know: refuse rather than guess
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return model.Unknown
	}
	return v * mult
}
