package main

// LED blackout. Own ALSA seq client writes straight to Push's Live port,
// same way push-manager's clearAllLEDs does (button CCs -> 0, pad notes -> 0).
// Live keeps its own LED state and only sends changes, so while the mode is
// on we repeat the blackout to win against late Live updates. LEDs in ledLit
// (Play, Session) stay lit.

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/alsaseq"
	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

var (
	ledMu  sync.Mutex
	ledSeq *alsaseq.Client
)

func setLEDClient(c *alsaseq.Client) {
	ledMu.Lock()
	ledSeq = c
	ledMu.Unlock()
}

// lit holds LEDs we want ON while everything else is dark: cc -> palette value.
var ledLit = map[byte]byte{}

// setLED lights (or clears) one button LED now and remembers it for the blackout loop.
func setLED(cc, val byte) {
	ledMu.Lock()
	defer ledMu.Unlock()
	if val == 0 {
		delete(ledLit, cc)
	} else {
		ledLit[cc] = val
	}
	if ledSeq != nil {
		_ = ledSeq.SendCC(pushLive.Addr(), 0, cc, int32(val))
	}
}

func resetLit() {
	ledMu.Lock()
	ledLit = map[byte]byte{}
	ledMu.Unlock()
}

// syncStateLEDs: Play is green while playing; Session is bright white when Back
// to Arrangement is available. the loop/punch buttons follow the transport.
func syncStateLEDs(playing, bta bool) {
	play, session := byte(ledWhite), byte(0)
	if playing {
		play = ledPlayGreen
	}
	if bta {
		session = ledWhite
	}
	setLED(byte(push3.CCPlay), play)
	setLED(byte(push3.CCSession), session)
}

// syncTransportLEDs: the three top-left buttons (Loop, Punch In, Punch Out)
// are white when off, yellow when on.
func syncTransportLEDs(loopOn, punchIn, punchOut bool) {
	v := func(on bool) byte {
		if on {
			return ledYellow
		}
		return ledWhite
	}
	setLED(byte(push3.CCScreenTop1), v(loopOn))
	setLED(byte(push3.CCScreenTop2), v(punchIn))
	setLED(byte(push3.CCScreenTop3), v(punchOut))
}

// syncNavLEDs: Page, Octave and the D-pad are always white while the mode is on.
func syncNavLEDs() {
	for _, cc := range []int{push3.CCPageLeft, push3.CCPageRight, push3.CCOctaveUp, push3.CCOctaveDown,
		push3.CCDPadLeft, push3.CCDPadRight, push3.CCDPadUp, push3.CCDPadDown, push3.CCDPadCenter} {
		setLED(byte(cc), ledWhite)
	}
}

// Palette indices chosen on the device.
const (
	ledPlayGreen = 126 // Play while playing
	ledWhite     = 120 // "on" for a greyscale button (Session, Page, Loop/Punch when off)
	ledYellow    = 8   // Loop/Punch when on
)

const (
	padNoteMin = 36
	padNoteMax = 99
)

// blackoutLEDs turns every button LED and every pad off.
func blackoutLEDs() {
	ledMu.Lock()
	defer ledMu.Unlock()
	if ledSeq == nil {
		return
	}
	// Every CC 1..127, not a list of known buttons: scene buttons, touch strip
	// cluster and anything else must go dark too. Non-LED CCs are ignored by Push.
	for cc := 1; cc <= 127; cc++ {
		_ = ledSeq.SendCC(pushLive.Addr(), 0, byte(cc), int32(ledLit[byte(cc)]))
	}
	for n := padNoteMin; n <= padNoteMax; n++ {
		_ = ledSeq.SendNote(pushLive.Addr(), 0, byte(n), 0)
	}
}

// pushLive: Push 3's Live port, where input comes from and LED messages go.
// Client number moves with sound cards (16 on card 0, 20 with loopback on
// card 0); runMIDI refreshes it before each subscribe.
var pushLive alsaseq.Push3

// ledActive: true while Arrangement Mode owns the LEDs.
var ledActive atomic.Bool

// burstDelays: Live repaints LEDs a moment after a command (Play, Back to
// Arrangement). Repeat the blackout right after, so the flicker is short.
var burstDelays = []time.Duration{0, 40 * time.Millisecond, 90 * time.Millisecond,
	160 * time.Millisecond, 260 * time.Millisecond, 420 * time.Millisecond,
	700 * time.Millisecond, 1100 * time.Millisecond}

// burstLEDs repeats the blackout for about a second. done (optional) runs at the
// end, to repaint the lit LEDs from the real state.
func burstLEDs(done func()) {
	go func() {
		var prev time.Duration
		for _, d := range burstDelays {
			time.Sleep(d - prev)
			prev = d
			if !ledActive.Load() {
				return
			}
			blackoutLEDs()
		}
		if done != nil && ledActive.Load() {
			done()
		}
	}()
}
