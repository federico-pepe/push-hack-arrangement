package main

// Touch layer: capacitive touches on Push 3's knobs and jog wheel show info
// over the arrangement while touched. Design: docs/touch-layer.md.
//
// Note On (velocity > 0) = touch, Note Off (or velocity 0) = release. The
// layer stays up for a short while after release, so a quick touch is still
// readable; touching again during that window keeps it up with no gap.

import (
	"sync"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

const touchHoldAfterRelease = 500 * time.Millisecond

// Touch notes this hack reacts to, named from core/push3 rather than
// restated as magic numbers.
const (
	noteEncoder1Touch = push3.NoteEncoder1Touch
	noteEncoder2Touch = push3.NoteEncoder2Touch
	noteJogTouch      = push3.NoteJogTouch
)

// touchOrder: the order lines are drawn in when more than one is touched.
var touchOrder = []uint8{noteJogTouch, noteEncoder1Touch, noteEncoder2Touch}

type touchLayer struct {
	mu    sync.Mutex
	held  map[uint8]bool
	until map[uint8]time.Time // release grace period, per note

	lines   func() map[uint8]string // note -> current text; called at render time
	changed func()                  // ask for a redraw
}

func newTouchLayer(lines func() map[uint8]string, changed func()) *touchLayer {
	return &touchLayer{held: map[uint8]bool{}, until: map[uint8]time.Time{}, lines: lines, changed: changed}
}

// onNote handles one touch note. Notes outside touchOrder are ignored.
func (t *touchLayer) onNote(note uint8, on bool) {
	if !isTouchNote(note) {
		return
	}
	t.mu.Lock()
	if on {
		t.held[note] = true
		delete(t.until, note)
	} else {
		delete(t.held, note)
		t.until[note] = time.Now().Add(touchHoldAfterRelease)
		// Ask for one more redraw once the grace period ends, so the layer
		// actually disappears (nothing else would trigger that frame).
		time.AfterFunc(touchHoldAfterRelease+20*time.Millisecond, t.changed)
	}
	t.mu.Unlock()
	t.changed()
}

func isTouchNote(note uint8) bool {
	for _, n := range touchOrder {
		if n == note {
			return true
		}
	}
	return false
}

// active: notes currently touched or still in their release grace period.
func (t *touchLayer) active() []uint8 {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	var out []uint8
	for _, n := range touchOrder {
		if t.held[n] {
			out = append(out, n)
			continue
		}
		if u, ok := t.until[n]; ok && now.Before(u) {
			out = append(out, n)
		}
	}
	return out
}

// text: the lines to draw, freshly computed, or nil if nothing is touched.
func (t *touchLayer) text() []string {
	active := t.active()
	if len(active) == 0 {
		return nil
	}
	lines := t.lines()
	out := make([]string, 0, len(active))
	for _, n := range active {
		if s, ok := lines[n]; ok {
			out = append(out, s)
		}
	}
	return out
}
