package main

import (
	"github.com/federico-pepe/ableton-push-hack/core/push3"
	"sync"
	"testing"
	"time"
)

func testInfo() touchInfo {
	return touchInfo{
		cols: []overlayCol{{col: 0, button: "Loop ON", name: "Loop Start", value: "1.1.1"}},
		text: map[uint8]string{
			noteTempoTouch: "Zoom", noteVolumeTouch: "View", noteJogTouch: "PLAYHEAD 1.1.1", noteEncoder1Touch: "LOOP START 1.1.1",
		},
	}
}

func waitShow() { time.Sleep(touchShowDelay + 20*time.Millisecond) }

func newTestTouchLayer() (*touchLayer, *int) {
	n := 0
	var mu sync.Mutex
	tl := newTouchLayer(testInfo, func() { mu.Lock(); n++; mu.Unlock() })
	return tl, &n
}

func TestEncoderTouchShowsColumns(t *testing.T) {
	tl, _ := newTestTouchLayer()
	if v := tl.view(); v.cols != nil || v.bar != nil {
		t.Fatal("nothing touched yet")
	}
	tl.onNote(noteEncoder1Touch, true)
	if v := tl.view(); v.cols != nil {
		t.Fatal("must wait the show delay")
	}
	waitShow()
	if v := tl.view(); len(v.cols) != 1 || v.bar != nil {
		t.Fatalf("got %+v", v)
	}
	tl.onNote(noteEncoder1Touch, false)
	tl.onNote(5, true) // any of the 8 knobs shows the header
	waitShow()
	if v := tl.view(); len(v.cols) != 1 {
		t.Fatalf("knob 6 must show the header: %+v", v)
	}
}

func TestJogShowsBarOnly(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onNote(noteJogTouch, true)
	v := tl.view()
	if len(v.bar) != 1 || v.cols != nil {
		t.Fatalf("got %+v", v)
	}
}

func TestTempoVolumeShowBarAtOnce(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onNote(noteVolumeTouch, true)
	tl.onNote(noteTempoTouch, true)
	v := tl.view() // no delay, no dim
	if len(v.bar) != 2 || v.bar[0] != "Zoom" || v.bar[1] != "View" || v.cols != nil {
		t.Fatalf("got %+v", v)
	}
}

func TestTouchStaysBrieflyAfterRelease(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onNote(noteJogTouch, true)
	tl.onNote(noteJogTouch, false)
	if v := tl.view(); len(v.bar) != 1 {
		t.Fatalf("must still show right after release: %+v", v)
	}
	time.Sleep(touchHoldAfterRelease + 50*time.Millisecond)
	if v := tl.view(); v.bar != nil {
		t.Fatalf("must be gone after the grace period: %+v", v)
	}
}

func TestTouchAgainDuringGraceCancelsTimeout(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onNote(noteEncoder2Touch, true)
	waitShow()
	tl.onNote(noteEncoder2Touch, false)
	time.Sleep(touchHoldAfterRelease / 2)
	tl.onNote(noteEncoder2Touch, true) // touched again before it expired
	time.Sleep(touchHoldAfterRelease/2 + 30*time.Millisecond)
	if v := tl.view(); len(v.cols) != 1 { // no new delay either: it was already showing
		t.Fatalf("re-touch must keep it up: %+v", v)
	}
}

func TestUnrelatedNoteIgnored(t *testing.T) {
	tl, calls := newTestTouchLayer()
	tl.onNote(99, true)
	if v := tl.view(); v.cols != nil || v.bar != nil {
		t.Fatal("note 99 is not a tracked touch")
	}
	if *calls != 0 {
		t.Fatal("must not even request a redraw for an unrelated note")
	}
}

func TestTurnHidesOverlayAndShowsBar(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onNote(noteEncoder1Touch, true)
	tl.onTurn(push3.CCEncoder1) // a turn inside the delay: bar at once, no overlay
	v := tl.view()
	if v.cols != nil || len(v.bar) != 1 || v.bar[0] != "LOOP START 1.1.1" {
		t.Fatalf("got %+v", v)
	}
	// a new touch brings the overlay back
	tl.onNote(noteEncoder1Touch, false)
	tl.onNote(noteEncoder1Touch, true)
	if v := tl.view(); v.cols != nil {
		t.Fatal("must wait the show delay")
	}
	waitShow()
	if v := tl.view(); len(v.cols) != 1 || v.bar != nil {
		t.Fatalf("new touch must show the overlay again: %+v", v)
	}
}

func TestTurnWithoutTouchIgnored(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onTurn(push3.CCEncoder1)
	tl.onNote(noteEncoder1Touch, true)
	waitShow()
	if v := tl.view(); len(v.cols) != 1 {
		t.Fatalf("a turn before the touch must not count: %+v", v)
	}
}

func TestButtonPressShowsColumns(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onButton(push3.CCScreenTop1, 127)
	if v := tl.view(); len(v.cols) != 1 {
		t.Fatalf("press must show the header: %+v", v)
	}
	tl.onButton(push3.CCScreenTop1, 0)
	time.Sleep(touchHoldAfterRelease + 50*time.Millisecond)
	if v := tl.view(); v.cols != nil {
		t.Fatalf("header must go after the grace period: %+v", v)
	}
	tl.onButton(push3.CCPlay, 127) // not a top button
	if v := tl.view(); v.cols != nil {
		t.Fatalf("other CC must not show it: %+v", v)
	}
}

func TestQuickTapNeverShows(t *testing.T) {
	tl, _ := newTestTouchLayer()
	tl.onNote(noteEncoder1Touch, true)
	tl.onNote(noteEncoder1Touch, false)
	waitShow()
	if v := tl.view(); v.cols != nil {
		t.Fatalf("a tap shorter than the delay must not show: %+v", v)
	}
}
