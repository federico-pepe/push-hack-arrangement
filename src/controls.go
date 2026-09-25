package main

// Control handling while Arrangement Mode is on.
//
//   Jog wheel        move the playhead (grid step by zoom, speed ramp)
//   Volume knob      scroll the view in time
//   Tempo knob       zoom; pressing it switches between time zoom and track zoom
//   Play             start / stop Live
//   Session          Back to Arrangement (Shift + Session = leave the mode)
//   D-pad            scroll; hold to repeat

import (
	"math"
	"sync"
	"time"
)

const (
	ccPlay = 85

	jogTargetPx = 10 // wanted screen distance of one jog tick, before the speed ramp

	toastFor = 1200 * time.Millisecond

	repeatDelay     = 450 * time.Millisecond // hold time before a D-pad button repeats
	repeatEvery     = 70 * time.Millisecond
	repeatFastAfter = 2 * time.Second // held this long: repeat twice as fast
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
	feedback  func(what string) // after a command that makes Live repaint LEDs

	mu      sync.Mutex
	lastJog time.Time
	hold    map[uint8]chan struct{} // D-pad buttons being held
	zoomV   bool                    // Tempo knob zooms tracks (true) or time (false)
}

func newControls(vc *viewCtl, st *store, isOn, shiftHeld func() bool, changed func(), send func(map[string]any), feedback func(string)) *controls {
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
		c.jog(decodeRel(val))
	case ccVolumeDial:
		c.vc.scrollT(decodeRel(val))
		c.changed()
	case ccTempoDial:
		c.mu.Lock()
		vertical := c.zoomV
		c.mu.Unlock()
		if vertical {
			c.vc.zoomV(decodeRel(val))
		} else {
			c.vc.zoomH(decodeRel(val))
		}
		c.changed()
	case ccTempoPress:
		if val > 0 {
			c.toggleZoom()
		}
	case ccPlay:
		if val > 0 {
			c.send(map[string]any{"t": "play_toggle"})
			c.feedback("play")
		}
	case ccSession:
		// Shift + Session is the mode chord (handled before we get here).
		if val > 0 && !c.shiftHeld() {
			c.send(map[string]any{"t": "bta"})
			c.feedback("bta")
		}
	case ccDPadUp, ccDPadDown, ccDPadLeft, ccDPadRight:
		c.dpad(cc, val)
	default:
		if c.vc.handleCC(cc, val) {
			c.changed()
		}
	}
}

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
	t, _ := c.st.pos()
	step := gridFor(c.vc.viewport(s).ppb)
	nt := jogTarget(t, step, ticks)
	c.st.setLocalTime(nt)
	c.send(map[string]any{"t": "set_time", "v": nt})
	c.vc.reveal(nt)
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
