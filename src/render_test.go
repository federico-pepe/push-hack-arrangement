package main

import "testing"

func TestTestFrameSize(t *testing.T) {
	b := renderTestFrame().Bounds()
	if b.Dx() != 960 || b.Dy() != 160 {
		t.Fatalf("got %v", b)
	}
}

func TestOSDKeepsSizeAndDoesNotMutateBase(t *testing.T) {
	base := renderTestFrame()
	before := append([]byte(nil), base.Pix...)
	out := renderWithOSD(base, "Arrangement Mode: ON")
	if out.Bounds() != base.Bounds() {
		t.Fatal("size changed")
	}
	for i := range before {
		if base.Pix[i] != before[i] {
			t.Fatal("base mutated")
		}
	}
}

func TestItoa(t *testing.T) {
	for n, want := range map[int]string{0: "0", 7: "7", 120: "120"} {
		if got := itoa(n); got != want {
			t.Errorf("itoa(%d)=%q want %q", n, got, want)
		}
	}
}
