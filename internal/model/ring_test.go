package model

import (
	"reflect"
	"testing"
)

func mkSample(free float64) CPUSample { return CPUSample{FreePct: free} }

func TestRingAppendAndSliceOrder(t *testing.T) {
	r := NewRing(3)
	if r.Len() != 0 || r.Cap() != 3 {
		t.Fatalf("new ring: len=%d cap=%d, want 0/3", r.Len(), r.Cap())
	}
	r.Append(mkSample(1))
	r.Append(mkSample(2))
	if got := r.FreePctSeries(); !reflect.DeepEqual(got, []float64{1, 2}) {
		t.Fatalf("partial series = %v, want [1 2]", got)
	}
}

func TestRingWrapAroundKeepsNewestInOrder(t *testing.T) {
	r := NewRing(3)
	for i := 1; i <= 5; i++ {
		r.Append(mkSample(float64(i)))
	}
	if r.Len() != 3 {
		t.Fatalf("len = %d, want 3 (capped)", r.Len())
	}
	// Oldest→newest should be the last three appended: 3,4,5.
	if got := r.FreePctSeries(); !reflect.DeepEqual(got, []float64{3, 4, 5}) {
		t.Fatalf("wrapped series = %v, want [3 4 5]", got)
	}
}

func TestRingZeroSizeIsUsable(t *testing.T) {
	r := NewRing(0) // clamped to 1
	r.Append(mkSample(9))
	r.Append(mkSample(10))
	if got := r.FreePctSeries(); !reflect.DeepEqual(got, []float64{10}) {
		t.Fatalf("series = %v, want [10]", got)
	}
}
