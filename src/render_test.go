package main

import "testing"

func TestTestFrameSize(t *testing.T) {
	b := renderTestFrame().Bounds()
	if b.Dx() != 960 || b.Dy() != 160 {
		t.Fatalf("got %v", b)
	}
}

func TestItoa(t *testing.T) {
	for n, want := range map[int]string{0: "0", 7: "7", 120: "120"} {
		if got := itoa(n); got != want {
			t.Errorf("itoa(%d)=%q want %q", n, got, want)
		}
	}
}
