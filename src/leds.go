package main

// LED blackout. Own ALSA seq client writes straight to Push's Live port,
// same way push-manager's clearAllLEDs does (button CCs -> 0, pad notes -> 0).
// Live keeps its own LED state and only sends changes, so while the mode is
// on we repeat the blackout to win against late Live updates. LEDs in ledLit
// (Play, Session) stay lit.

import (
	"sync"

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
		_ = ledSeq.SendCC(push3Live, 0, cc, int32(val))
	}
}

func resetLit() {
	ledMu.Lock()
	ledLit = map[byte]byte{}
	ledMu.Unlock()
}

// syncStateLEDs: Play is green while playing; Session is bright white when Back to Arrangement is available.
func syncStateLEDs(playing, bta bool) {
	play, session := byte(ledPlayWhite), byte(0)
	if playing {
		play = ledPlayGreen
	}
	if bta {
		session = ledSessionWhite
	}
	setLED(byte(push3.CCPlay), play)
	setLED(byte(push3.CCSession), session)
}

// Palette indices chosen on the device.
const (
	ledPlayGreen    = 126 // Play while playing
	ledPlayWhite    = 120 // Play when stopped
	ledSessionWhite = 120 // Back to Arrangement available (this button has a greyscale LED)
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
		_ = ledSeq.SendCC(push3Live, 0, byte(cc), int32(ledLit[byte(cc)]))
	}
	for n := padNoteMin; n <= padNoteMax; n++ {
		_ = ledSeq.SendNote(push3Live, 0, byte(n), 0)
	}
}

// push3Live: Push 3's Live port, where LED messages go.
var push3Live = alsaseq.Addr{Client: alsaseq.Push3ClientDefault, Port: alsaseq.Push3PortDefault}
