package main

import "testing"

func TestBBSPosition(t *testing.T) {
	cases := []struct {
		beats, bpb float64
		want       string
	}{
		{0, 4, "1.1.1"},
		{384, 4, "97.1.1"}, // matches the real set's saved LoopStart
		{4, 4, "2.1.1"},
		{4.25, 4, "2.1.2"}, // 1/16 beat into bar 2, beat 1
		{-3, 4, "1.1.1"},   // clamps at 0
	}
	for _, c := range cases {
		if got := bbsPosition(c.beats, c.bpb); got != c.want {
			t.Errorf("bbsPosition(%v,%v)=%q want %q", c.beats, c.bpb, got, c.want)
		}
	}
}

func TestBBSLength(t *testing.T) {
	cases := []struct {
		beats, bpb float64
		want       string
	}{
		{0, 4, "0.0.0"},
		{64, 4, "16.0.0"}, // matches the user's "16 bar loop" example
		{4, 4, "1.0.0"},
		{4.25, 4, "1.0.1"},
	}
	for _, c := range cases {
		if got := bbsLength(c.beats, c.bpb); got != c.want {
			t.Errorf("bbsLength(%v,%v)=%q want %q", c.beats, c.bpb, got, c.want)
		}
	}
}

func TestSplitBBSDefaultsBpb(t *testing.T) {
	if bar, beat, six := splitBBS(4, 0); bar != 1 || beat != 0 || six != 0 {
		t.Fatalf("bpb<=0 must default to 4: got %d %d %d", bar, beat, six)
	}
}
