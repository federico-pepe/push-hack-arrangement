package main

// LED blackout. Own ALSA seq client writes straight to Push's Live port,
// same way push-manager's clearAllLEDs does (button CCs -> 0, pad notes -> 0).
// Live keeps its own LED state and only sends changes, so while the mode is
// on we repeat the blackout to win against late Live updates.

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

// Button CCs that carry an LED (same set as push-manager).
var ledButtonCCs = buildLEDButtonCCs()

func buildLEDButtonCCs() []byte {
	var l []byte
	for n := 0; n < 8; n++ {
		l = append(l, push3.CCScreenTopN(n), push3.CCScreenBotN(n))
	}
	for _, cc := range []int{
		push3.CCSet, push3.CCSettings, push3.CCHelp, push3.CCUserMode,
		push3.CCDeviceView, push3.CCMixerView, push3.CCClipView, push3.CCSessionView,
		push3.CCShift, push3.CCSelect,
		push3.CCUndo, push3.CCSave, push3.CCAdd, push3.CCSwap,
		push3.CCLock, push3.CCStopClips, push3.CCMute, push3.CCSolo, push3.CCSelectMain,
		push3.CCTapTempo, push3.CCMetronome, push3.CCQuantize, push3.CCFixedLength,
		push3.CCAutomate, push3.CCNew, push3.CCCapture, push3.CCRecord, push3.CCPlay,
		push3.CCRepeat, push3.CCAccent, push3.CCScale, push3.CCLayout, push3.CCNote, push3.CCSession,
		push3.CCDoubleLoop, push3.CCDuplicate, push3.CCConvert, push3.CCDelete,
		push3.CCOctaveUp, push3.CCOctaveDown, push3.CCPageLeft, push3.CCPageRight,
		push3.CCDPadUp, push3.CCDPadRight, push3.CCDPadDown, push3.CCDPadLeft, push3.CCDPadCenter,
		push3.CCJogPress, push3.CCJogClickLeft, push3.CCJogClickRight,
	} {
		l = append(l, byte(cc))
	}
	return l
}

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
	dst := alsaseq.Addr{Client: alsaseq.Push3ClientDefault, Port: alsaseq.Push3PortDefault}
	for _, cc := range ledButtonCCs {
		_ = ledSeq.SendCC(dst, 0, cc, 0)
	}
	for n := padNoteMin; n <= padNoteMax; n++ {
		_ = ledSeq.SendNote(dst, 0, byte(n), 0)
	}
}
