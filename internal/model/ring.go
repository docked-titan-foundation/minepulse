package model

// Ring is a fixed-capacity circular buffer of CPUSamples used to draw a node's
// sparkline over the session. Append is O(1); Slice returns oldest→newest.
type Ring struct {
	buf   []CPUSample
	start int // index of the oldest element
	count int
}

// NewRing returns a ring holding up to size samples. size <= 0 yields capacity 1.
func NewRing(size int) *Ring {
	if size <= 0 {
		size = 1
	}
	return &Ring{buf: make([]CPUSample, size)}
}

// Append adds a sample, overwriting the oldest once full.
func (r *Ring) Append(s CPUSample) {
	if r.count < len(r.buf) {
		r.buf[(r.start+r.count)%len(r.buf)] = s
		r.count++
		return
	}
	r.buf[r.start] = s
	r.start = (r.start + 1) % len(r.buf)
}

// Len is the number of samples currently held.
func (r *Ring) Len() int { return r.count }

// Cap is the maximum number of samples the ring can hold.
func (r *Ring) Cap() int { return len(r.buf) }

// Slice returns the samples oldest→newest.
func (r *Ring) Slice() []CPUSample {
	out := make([]CPUSample, r.count)
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(r.start+i)%len(r.buf)]
	}
	return out
}

// FreePctSeries returns just the FreePct values oldest→newest (for sparklines).
func (r *Ring) FreePctSeries() []float64 {
	out := make([]float64, r.count)
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(r.start+i)%len(r.buf)].FreePct
	}
	return out
}
