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

// touchShowDelay: a knob touch shows the overlay only after this long, so a
// quick touch, or a turn that starts at once, never flashes it. Tempo, Volume, the jog
// wheel and the top buttons show at once.
const touchShowDelay = 150 * time.Millisecond

func delayed(n uint8) bool { return isEncoderNote(n) }

// Touch notes this hack reacts to, named from core/push3 rather than
// restated as magic numbers.
const (
	noteEncoder1Touch = push3.NoteEncoder1Touch
	noteEncoder2Touch = push3.NoteEncoder2Touch
	noteJogTouch      = push3.NoteJogTouch
	noteVolumeTouch   = push3.NoteVolumeTouch
	noteTempoTouch    = push3.NoteTempoTouch
)

// touchOrder: the notes this layer tracks. Encoders 1-8 are notes 0-7.
var touchOrder = []uint8{noteJogTouch, noteTempoTouch, noteVolumeTouch, 0, 1, 2, 3, 4, 5, 6, 7,
	noteButton1, noteButton2, noteButton3}

// Pseudo notes for the top buttons 1-3 (Loop, Punch In, Punch Out). A real MIDI
// note is 0-127, so these never clash with a touch note. A press works like a
// touch: the header shows while the button is down and for a moment after.
const (
	noteButton1 uint8 = 128 + iota
	noteButton2
	noteButton3
)

func isButtonNote(n uint8) bool { return n >= noteButton1 && n <= noteButton3 }

func isEncoderNote(n uint8) bool { return n <= 7 }

// overlayCol: one column of the touch overlay. Like Live's own header, it
// shows the top button's function (button), then the encoder's name and value.
type overlayCol struct {
	col         int
	button      string // top button label, e.g. "Loop"
	toggle      bool   // the button is an on/off switch: draw an On/Off box after the label
	buttonOn    bool
	name, value string
}

// touchInfo: everything the layer can show, computed fresh at render time.
type touchInfo struct {
	cols []overlayCol     // shown together when any encoder is touched
	text map[uint8]string // one line per control: value text (Tempo, Volume, jog, Encoders 1/2)
}

type touchLayer struct {
	mu    sync.Mutex
	held  map[uint8]bool
	until map[uint8]time.Time // release grace period, per note
	since map[uint8]time.Time // touch start, per note
	moved map[uint8]bool      // turned since the touch began: overlay gives way to the bar

	info    func() touchInfo // called at render time
	changed func()           // ask for a redraw
}

func newTouchLayer(info func() touchInfo, changed func()) *touchLayer {
	return &touchLayer{held: map[uint8]bool{}, moved: map[uint8]bool{}, since: map[uint8]time.Time{}, until: map[uint8]time.Time{}, info: info, changed: changed}
}

// onButton: a top button 1-3 CC (127 down, 0 up). Other CCs are ignored.
func (t *touchLayer) onButton(cc, val uint8) {
	switch cc {
	case push3.CCScreenTop1:
		t.onNote(noteButton1, val > 0)
	case push3.CCScreenTop2:
		t.onNote(noteButton2, val > 0)
	case push3.CCScreenTop3:
		t.onNote(noteButton3, val > 0)
	}
}

// onNote handles one touch note. Notes outside touchOrder are ignored.
func (t *touchLayer) onNote(note uint8, on bool) {
	if !isTouchNote(note) {
		return
	}
	t.mu.Lock()
	if on {
		t.held[note] = true
		if u, ok := t.until[note]; !ok || !time.Now().Before(u) {
			t.since[note] = time.Now() // not a re-touch inside the hold: wait the delay again
		}
		delete(t.until, note)
		delete(t.moved, note)
		if delayed(note) {
			time.AfterFunc(touchShowDelay+10*time.Millisecond, t.changed)
		}
	} else {
		delete(t.held, note)
		if delayed(note) && time.Since(t.since[note]) < touchShowDelay {
			t.mu.Unlock() // a quick tap: it never showed, so no hold-after-release
			t.changed()
			return
		}
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

// onTurn: the control behind a CC was turned. If its touch is on, the overlay
// steps aside and the value shows in the bottom bar instead. Returns at once
// for a CC with no touch note.
func (t *touchLayer) onTurn(cc uint8) {
	var note uint8
	switch {
	case cc >= push3.CCEncoder1 && cc <= push3.CCEncoder8:
		note = cc - push3.CCEncoder1
	case cc == push3.CCTempo:
		note = noteTempoTouch
	case cc == push3.CCVolume:
		note = noteVolumeTouch
	default:
		return
	}
	t.mu.Lock()
	_, grace := t.until[note]
	if t.held[note] || grace {
		t.moved[note] = true
	}
	t.mu.Unlock()
}

// touchView: what to draw right now.
type touchView struct {
	cols []overlayCol // header columns (dim the screen)
	bar  []string     // bottom bar: values of controls being turned, and the jog wheel
}

// view: freshly computed, zero value if nothing is touched. A touched control
// that has been turned shows its value in the bar and no overlay at all.
func (t *touchLayer) view() touchView {
	active := t.active()
	var v touchView
	if len(active) == 0 {
		return v
	}
	info := t.info()
	t.mu.Lock()
	anyMoved := false
	for _, n := range active {
		if t.moved[n] {
			anyMoved = true
		}
	}
	movedNow := map[uint8]bool{}
	for n := range t.moved {
		movedNow[n] = true
	}
	since := map[uint8]time.Time{}
	for n, ts := range t.since {
		since[n] = ts
	}
	t.mu.Unlock()
	for _, n := range active {
		txt, has := info.text[n]
		if delayed(n) && !movedNow[n] && time.Since(since[n]) < touchShowDelay {
			continue // not shown yet
		}
		switch {
		case n == noteJogTouch:
			if has {
				v.bar = append(v.bar, txt)
			}
		case movedNow[n]:
			if has {
				v.bar = append(v.bar, txt)
			}
		case anyMoved: // another control is being turned: keep the screen clear
		case isEncoderNote(n) || isButtonNote(n):
			v.cols = info.cols
		default: // Tempo, Volume: the bar, no dim
			if has {
				v.bar = append(v.bar, txt)
			}
		}
	}
	return v
}
