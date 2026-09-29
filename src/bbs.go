package main

// Bars.Beats.16th formatting, same style as Live's own transport display.

import (
	"fmt"
	"math"
)

// bbsPosition formats a beat position (0 = the very start) as a 1-based
// "bar.beat.16th", e.g. 0 -> "1.1.1", 384 beats at 4/4 -> "97.1.1".
func bbsPosition(beats, bpb float64) string {
	bar, beat, six := splitBBS(beats, bpb)
	return fmt.Sprintf("%d.%d.%d", bar+1, beat+1, six+1)
}

// bbsLength formats a beat length (a duration, not a position) as a 0-based
// "bars.beats.16th", e.g. 64 beats at 4/4 -> "16.0.0".
func bbsLength(beats, bpb float64) string {
	bar, beat, six := splitBBS(beats, bpb)
	return fmt.Sprintf("%d.%d.%d", bar, beat, six)
}

// splitBBS breaks a beat count into (whole bars, beat within the bar, 16th
// within the beat), all 0-based. Negative input clamps to 0.
func splitBBS(beats, bpb float64) (bar, beat, six int) {
	if bpb <= 0 {
		bpb = 4
	}
	if beats < 0 {
		beats = 0
	}
	b := math.Floor(beats / bpb)
	rem := beats - b*bpb
	be := math.Floor(rem)
	six64 := math.Floor((rem - be) * 4)
	return int(b), int(be), int(six64)
}
