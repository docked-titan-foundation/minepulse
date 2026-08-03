package render

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

// Sparkline renders values as a unicode block sparkline scaled to [lo,hi].
// width caps the number of (most recent) points shown.
func Sparkline(values []float64, lo, hi float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}
	if len(values) > width {
		values = values[len(values)-width:]
	}
	span := hi - lo
	if span <= 0 {
		span = 1
	}
	out := make([]rune, len(values))
	for i, v := range values {
		frac := (v - lo) / span
		switch {
		case frac < 0:
			frac = 0
		case frac > 1:
			frac = 1
		}
		idx := int(frac * float64(len(sparkRunes)-1))
		out[i] = sparkRunes[idx]
	}
	return string(out)
}
