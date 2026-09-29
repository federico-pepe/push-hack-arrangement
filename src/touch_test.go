package main

import (
	"sync"
	"testing"
	"time"
)

func newTestTouchLayer(lines map[uint8]string) (*touchLayer, *int) {
	n := 0
	var mu sync.Mutex
	tl := newTouchLayer(func() map[uint8]string { return lines },
		func() { mu.Lock(); n++; mu.Unlock() })
	return tl, &n
}

func TestTouchShowsWhileHeld(t *testing.T) {
	tl, _ := newTestTouchLayer(map[uint8]string{noteEncoder1Touch: "A"})
	if tl.text() != nil {
		t.Fatal("nothing touched yet")
	}
	tl.onNote(noteEncoder1Touch, true)
	if got := tl.text(); len(got) != 1 || got[0] != "A" {
		t.Fatalf("got %v", got)
	}
}

func TestTouchStaysBrieflyAfterRelease(t *testing.T) {
	tl, _ := newTestTouchLayer(map[uint8]string{noteJogTouch: "T"})
	tl.onNote(noteJogTouch, true)
	tl.onNote(noteJogTouch, false)
	if got := tl.text(); len(got) != 1 {
		t.Fatalf("must still show right after release: %v", got)
	}
	time.Sleep(touchHoldAfterRelease + 50*time.Millisecond)
	if got := tl.text(); got != nil {
		t.Fatalf("must be gone after the grace period: %v", got)
	}
}

func TestTouchAgainDuringGraceCancelsTimeout(t *testing.T) {
	tl, _ := newTestTouchLayer(map[uint8]string{noteEncoder2Touch: "L"})
	tl.onNote(noteEncoder2Touch, true)
	tl.onNote(noteEncoder2Touch, false)
	time.Sleep(touchHoldAfterRelease / 2)
	tl.onNote(noteEncoder2Touch, true) // touched again before it expired
	time.Sleep(touchHoldAfterRelease/2 + 30*time.Millisecond)
	if got := tl.text(); len(got) != 1 {
		t.Fatalf("re-touch must keep it up: %v", got)
	}
}

func TestTwoTouchesShowBothInOrder(t *testing.T) {
	tl, _ := newTestTouchLayer(map[uint8]string{
		noteJogTouch:      "PLAYHEAD",
		noteEncoder1Touch: "LOOP START",
	})
	tl.onNote(noteEncoder1Touch, true)
	tl.onNote(noteJogTouch, true)
	got := tl.text()
	if len(got) != 2 || got[0] != "PLAYHEAD" || got[1] != "LOOP START" {
		t.Fatalf("got %v, want jog before encoder1 (touchOrder)", got)
	}
}

func TestUnrelatedNoteIgnored(t *testing.T) {
	tl, calls := newTestTouchLayer(map[uint8]string{99: "X"})
	tl.onNote(99, true)
	if tl.text() != nil {
		t.Fatal("note 99 is not a tracked touch")
	}
	if *calls != 0 {
		t.Fatal("must not even request a redraw for an unrelated note")
	}
}
