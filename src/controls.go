package main

// Control handling while Arrangement Mode is on.
//
//   Jog wheel        move the scrub cursor -- local only, Live's playback is
//                    untouched (no audible jump). See jog()'s doc.
//   Shift + jog      the old way: really seek Live (current_song_time moves,
//                    audible if playing)
//   Jog wheel press  set the play-start anchor at the cursor/playhead (Shift +
//                    press clears it)
//   Volume knob      scroll the view in time
//   Tempo knob       zoom; pressing it switches between time zoom and track zoom
//   Play             start / stop Live -- from the anchor, if one is set
//   Session          Back to Arrangement (Shift + Session is the mode chord)
//   Page < / >       jump to the previous / next locator (a real seek)
//   Top button 1     Loop on/off
//   Top button 2     Punch In on/off
//   Top button 3     Punch Out on/off
//   Encoder 1        loop start, in bars (Shift = a precise 1/16 beat step)
//   Encoder 2        loop length, in bars (Shift = a precise 1/16 beat step)
//   D-pad            scroll; hold to repeat

import (
	"math"
	"sync"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/gfx/text"
	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

// CCs, named from core/push3 rather than restated as magic numbers.
const (
	ccPlay      = push3.CCPlay
	ccJogPress  = push3.CCJogPress   // jog wheel click: set/clear the play-start anchor
	ccPageLeft  = push3.CCPageLeft   // previous locator
	ccPageRight = push3.CCPageRight  // next locator
	ccTopLoop   = push3.CCScreenTop1 // top button 1: Loop on/off
	ccTopPunchI = push3.CCScreenTop2 // top button 2: Punch In
	ccTopPunchO = push3.CCScreenTop3 // top button 3: Punch Out
	ccEncoder1  = push3.CCEncoder1   // loop start
	ccEncoder2  = push3.CCEncoder2   // loop length

	jogTargetPx = 10 // wanted screen distance of one jog tick, before the speed ramp

	toastFor = 1200 * time.Millisecond

	repeatDelay     = 450 * time.Millisecond // hold time before a D-pad button repeats
	repeatEvery     = 70 * time.Millisecond
	repeatFastAfter = 2 * time.Second // held this long: repeat twice as fast

	loopStepBars      = 1.0  // bars per encoder detent
	loopStepFineBeats = 0.25 // Shift: a precise 1/16 beat step, not a coarser one
)

// jogGrid: musical steps, in beats. 1/64 .. 32 beats.
var jogGrid = []float64{1.0 / 64, 1.0 / 32, 1.0 / 16, 1.0 / 8, 1.0 / 4, 1.0 / 2, 1, 2, 4, 8, 16, 32}

// gridFor picks the step for a zoom: the smallest grid value that is at least
// jogTargetPx wide on screen.
func gridFor(ppb float64) float64 {
	want := jogTargetPx / ppb
	for _, g := range jogGrid {
		if g >= want {
			return g
		}
	}
	return jogGrid[len(jogGrid)-1]
}

// jogTarget returns the new time after `ticks` grid steps (sign = direction).
// Off-grid: the first step lands on the nearest grid line in the direction of travel.
func jogTarget(t, step float64, ticks int) float64 {
	if ticks == 0 {
		return t
	}
	base := t / step
	r := math.Round(base)
	var n float64
	switch {
	case math.Abs(base-r) < 1e-6:
		n = r + float64(ticks)
	case ticks > 0:
		n = math.Ceil(base) + float64(ticks-1)
	default:
		n = math.Floor(base) + float64(ticks+1)
	}
	return math.Max(0, n*step)
}

// jogMultiplier: faster spin (shorter gap between messages) = more steps per tick.
func jogMultiplier(gap time.Duration) int {
	switch {
	case gap < 25*time.Millisecond:
		return 8
	case gap < 40*time.Millisecond:
		return 4
	case gap < 60*time.Millisecond:
		return 2
	}
	return 1
}

type controls struct {
	vc        *viewCtl
	st        *store
	isOn      func() bool
	shiftHeld func() bool
	changed   func() // request a redraw
	send      func(map[string]any)
	feedback  func(what string, on bool) // after a command that makes Live repaint LEDs

	mu      sync.Mutex
	lastJog time.Time
	hold    map[uint8]chan struct{} // D-pad buttons being held
	zoomV   bool                    // Tempo knob zooms tracks (true) or time (false)
}

func newControls(vc *viewCtl, st *store, isOn, shiftHeld func() bool, changed func(),
	send func(map[string]any), feedback func(string, bool)) *controls {
	return &controls{vc: vc, st: st, isOn: isOn, shiftHeld: shiftHeld, changed: changed, send: send,
		feedback: feedback, hold: map[uint8]chan struct{}{}}
}

// onCC handles one control-surface CC.
func (c *controls) onCC(cc, val uint8) {
	if !c.isOn() {
		return
	}
	switch cc {
	case ccJog:
		c.jog(push3.DecodeRel(val))
	case ccJogPress:
		if val > 0 {
			c.jogPress()
		}
	case ccVolumeDial:
		c.vc.scrollT(push3.DecodeRel(val))
		c.changed()
	case ccTempoDial:
		c.mu.Lock()
		vertical := c.zoomV
		c.mu.Unlock()
		if vertical {
			c.vc.zoomV(push3.DecodeRel(val))
		} else {
			c.vc.zoomH(push3.DecodeRel(val))
		}
		c.changed()
	case ccTempoPress:
		if val > 0 {
			c.toggleZoom()
		}
	case ccPlay:
		if val > 0 {
			c.play()
		}
	case ccSession:
		// Shift + Session is the mode chord (handled before we get here).
		if val > 0 && !c.shiftHeld() {
			c.send(map[string]any{"t": "bta"})
			c.feedback("bta", false) // BTA always clears the override
			c.changed()
		}
	case ccPageLeft:
		if val > 0 {
			c.locatorNav(false)
		}
	case ccPageRight:
		if val > 0 {
			c.locatorNav(true)
		}
	case ccTopLoop:
		if val > 0 {
			c.toggleLoop()
		}
	case ccTopPunchI:
		if val > 0 {
			c.togglePunchIn()
		}
	case ccTopPunchO:
		if val > 0 {
			c.togglePunchOut()
		}
	case ccEncoder1:
		c.adjustLoop(true, push3.DecodeRel(val))
	case ccEncoder2:
		c.adjustLoop(false, push3.DecodeRel(val))
	case ccDPadUp, ccDPadDown, ccDPadLeft, ccDPadRight:
		c.dpad(cc, val)
	default:
		if c.vc.handleCC(cc, val) {
			c.changed()
		}
	}
}

// jog moves the scrub cursor -- a local-only marker, separate from Live's real
// playhead. Turning it never seeks Live, so playback (if running) is not
// disturbed and nothing audibly jumps. Shift + jog is the old, real-seek
// behaviour (current_song_time actually moves): use it when you DO want to cue
// to a spot. See cursorOrPlayhead's doc for how the two markers relate.
func (c *controls) jog(delta int) {
	if delta == 0 {
		return
	}
	s, _, _ := c.st.get()
	if s == nil {
		return
	}
	now := time.Now()
	c.mu.Lock()
	gap := now.Sub(c.lastJog)
	c.lastJog = now
	c.mu.Unlock()

	ticks := delta * jogMultiplier(gap)
	step := gridFor(c.vc.viewport(s).ppb)

	if c.shiftHeld() {
		t, _ := c.st.pos()
		nt := jogTarget(t, step, ticks)
		c.st.setLocalTime(nt) // a real seek -- also drops any scrub cursor
		c.send(map[string]any{"t": "set_time", "v": nt})
		c.vc.reveal(nt)
		c.changed()
		return
	}

	nt := jogTarget(c.cursorOrPlayhead(), step, ticks)
	c.st.setCursor(nt)
	c.vc.reveal(nt)
	c.changed()
}

// cursorOrPlayhead: the scrub cursor if one is active, else the real playhead
// -- the reference point for "where the user is currently pointing".
func (c *controls) cursorOrPlayhead() float64 {
	if t, ok := c.st.getCursor(); ok {
		return t
	}
	t, _ := c.st.pos()
	return t
}

// jogPress: set the play-start anchor at the cursor/playhead. Shift + press
// clears it, so Play starts from wherever the real playhead is instead.
func (c *controls) jogPress() {
	if c.shiftHeld() {
		c.st.clearAnchor()
		c.vc.setToast("ANCHOR CLEARED", toastFor)
	} else {
		c.st.setAnchor(c.cursorOrPlayhead())
		c.vc.setToast("ANCHOR SET", toastFor)
	}
	c.changed()
}

// play starts or stops Live. Starting jumps to the anchor first, if one is set.
func (c *controls) play() {
	playing, _ := c.st.state()
	if !playing {
		if t, ok := c.st.getAnchor(); ok {
			c.st.setLocalTime(t)
			c.send(map[string]any{"t": "set_time", "v": t})
			c.vc.reveal(t)
		}
	}
	c.send(map[string]any{"t": "play_toggle"})
	c.feedback("play", !playing)
}

// locatorNav jumps to the next/previous locator from the current playhead (no
// wrap: past the last locator, it just stays on the last one).
func (c *controls) locatorNav(forward bool) {
	s, _, _ := c.st.get()
	if s == nil || len(s.Locators) == 0 {
		return
	}
	t := c.cursorOrPlayhead()
	var target *Locator
	if forward {
		for i := range s.Locators {
			if s.Locators[i].Time > t+1e-6 {
				target = &s.Locators[i]
				break
			}
		}
		if target == nil {
			target = &s.Locators[len(s.Locators)-1]
		}
	} else {
		for i := len(s.Locators) - 1; i >= 0; i-- {
			if s.Locators[i].Time < t-1e-6 {
				target = &s.Locators[i]
				break
			}
		}
		if target == nil {
			target = &s.Locators[0]
		}
	}
	c.st.setLocalTime(target.Time)
	c.st.setSelLocator(target.Time)
	c.send(map[string]any{"t": "set_time", "v": target.Time})
	c.vc.reveal(target.Time)
	label := target.Name
	if label == "" {
		label = "LOCATOR"
	}
	c.vc.setToast(text.Truncate(label, 12)+" "+bbsPosition(target.Time, s.bpb()), toastFor)
	c.changed()
}

// toggleLoop / togglePunchIn / togglePunchOut: flip the flag in Live and light
// the LED at once (the actual flag is confirmed a moment later over "pos").
func (c *controls) toggleLoop() {
	on, _, _, _, _ := c.st.transport()
	c.send(map[string]any{"t": "loop_toggle"})
	c.feedback("loop", !on)
	c.vc.setToast(onOff("LOOP", !on), toastFor)
	c.changed()
}

func (c *controls) togglePunchIn() {
	_, in, _, _, _ := c.st.transport()
	c.send(map[string]any{"t": "punch_in_toggle"})
	c.feedback("punch_in", !in)
	c.vc.setToast(onOff("PUNCH IN", !in), toastFor)
	c.changed()
}

func (c *controls) togglePunchOut() {
	_, _, out, _, _ := c.st.transport()
	c.send(map[string]any{"t": "punch_out_toggle"})
	c.feedback("punch_out", !out)
	c.vc.setToast(onOff("PUNCH OUT", !out), toastFor)
	c.changed()
}

func onOff(label string, on bool) string {
	if on {
		return label + " ON"
	}
	return label + " OFF"
}

// adjustLoop moves the loop start (isStart) or length by one bar per detent.
// Shift switches to a precise 1/16 beat step instead -- finer, not coarser,
// for lining the loop up exactly.
func (c *controls) adjustLoop(isStart bool, delta int) {
	if delta == 0 {
		return
	}
	s, _, _ := c.st.get()
	if s == nil {
		return
	}
	step := s.bpb() * loopStepBars
	if c.shiftHeld() {
		step = loopStepFineBeats
	}
	_, _, _, start, length := c.st.transport()
	if isStart {
		start = math.Max(0, start+float64(delta)*step)
	} else {
		length = math.Max(step, length+float64(delta)*step)
	}
	c.st.setLocalLoop(start, length)
	c.send(map[string]any{"t": "set_loop", "start": start, "length": length})
	c.changed()
}

// toggleZoom switches the Tempo knob between time zoom and track zoom and says so on screen.
func (c *controls) toggleZoom() {
	c.mu.Lock()
	c.zoomV = !c.zoomV
	msg := "ZOOM: TIME"
	if c.zoomV {
		msg = "ZOOM: TRACKS"
	}
	c.mu.Unlock()
	c.vc.setToast(msg, toastFor)
	c.changed()
	time.AfterFunc(toastFor+50*time.Millisecond, c.changed) // redraw once it expires
}

// dpad: act on press, then repeat while held.
func (c *controls) dpad(cc, val uint8) {
	c.mu.Lock()
	stop, held := c.hold[cc]
	if val == 0 {
		if held {
			close(stop)
			delete(c.hold, cc)
		}
		c.mu.Unlock()
		return
	}
	if held { // already repeating
		c.mu.Unlock()
		return
	}
	stop = make(chan struct{})
	c.hold[cc] = stop
	c.mu.Unlock()

	c.dpadAction(cc)
	go func() {
		start := time.Now()
		select {
		case <-stop:
			return
		case <-time.After(repeatDelay):
		}
		for {
			if !c.isOn() {
				return
			}
			c.dpadAction(cc)
			every := repeatEvery
			if time.Since(start) > repeatFastAfter {
				every /= 2
			}
			select {
			case <-stop:
				return
			case <-time.After(every):
			}
		}
	}()
}

func (c *controls) dpadAction(cc uint8) {
	switch cc {
	case ccDPadUp:
		c.vc.scrollTracks(-1)
	case ccDPadDown:
		c.vc.scrollTracks(+1)
	case ccDPadLeft:
		c.vc.scrollPage(-1)
	case ccDPadRight:
		c.vc.scrollPage(+1)
	}
	c.changed()
}

// releaseAll stops every D-pad repeat (mode turned off).
func (c *controls) releaseAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for cc, ch := range c.hold {
		close(ch)
		delete(c.hold, cc)
	}
}
