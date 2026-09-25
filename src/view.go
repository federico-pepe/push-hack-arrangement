package main

// View state: zoom and scroll. All input handlers land here.
//
// View controls (while Arrangement Mode is on, MIDI intercept hides them from Live):
//   D-pad up/down       scroll tracks       D-pad left/right   scroll a quarter screen
// Dials, jog wheel, Play and Session are handled in controls.go.

import (
	"math"
	"sync"
	"time"
)

const (
	ccTempoDial  = 14
	ccTempoPress = 15 // pressing the Tempo knob in
	ccVolumeDial = 79
	ccJog        = 70
	ccDPadLeft   = 44
	ccDPadRight  = 45
	ccDPadUp     = 46
	ccDPadDown   = 47

	zoomStep      = 1.06            // per encoder tick
	maxPPB        = 300             // px per beat, most zoomed in
	jogPxPerTick  = 24              // Shift + jog: view scroll
	followHold    = 2 * time.Second // after a manual move, do not follow the playhead
	gutterW       = 110             // track-name column
	namesMinLane  = 13              // lane height (px) where names fit
	minSongBeats  = 16
	songTailBeats = 4
)

type viewCtl struct {
	mu       sync.Mutex
	get      func() *Set
	ready    bool
	x0       float64 // time at left edge of the clip area (beats)
	ppb      float64 // pixels per beat
	laneH    float64 // pixels per track lane
	first    float64 // first visible track (fractional)
	lastUser time.Time

	toast      string // short message drawn on the view
	toastUntil time.Time
}

func newViewCtl(get func() *Set) *viewCtl { return &viewCtl{get: get} }

func totalBeats(s *Set) float64 {
	t := s.Length + songTailBeats
	if t < minSongBeats {
		t = minSongBeats
	}
	return t
}

func fitPPB(s *Set) float64 { return float64(screenW) / totalBeats(s) }

func laneAvail() float64 { return float64(screenH - rulerH - 1) }

func fitLane(s *Set) float64 {
	n := len(s.Tracks)
	if n == 0 {
		return laneAvail()
	}
	return laneAvail() / float64(n)
}

// clamp keeps every value in range for set s; also sets the defaults on first use.
func (c *viewCtl) clamp(s *Set) {
	if !c.ready {
		c.ppb, c.laneH = fitPPB(s), fitLane(s)
		c.ready = true
	}
	c.ppb = math.Max(fitPPB(s), math.Min(maxPPB, c.ppb))
	c.laneH = math.Max(fitLane(s), math.Min(laneAvail(), c.laneH))
	span := c.span()
	c.x0 = math.Max(0, math.Min(c.x0, math.Max(0, totalBeats(s)-span)))
	visible := laneAvail() / c.laneH
	c.first = math.Max(0, math.Min(c.first, math.Max(0, float64(len(s.Tracks))-visible)))
}

func (c *viewCtl) gutter() int {
	if c.laneH >= namesMinLane {
		return gutterW
	}
	return 0
}

// span: beats visible in the clip area.
func (c *viewCtl) span() float64 { return float64(screenW-c.gutter()) / c.ppb }

// viewport returns the clamped view for s.
func (c *viewCtl) viewport(s *Set) viewport {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	vp := viewport{x0: c.x0, ppb: c.ppb, laneH: c.laneH, first: c.first, gutter: c.gutter()}
	if time.Now().Before(c.toastUntil) {
		vp.toast = c.toast
	}
	return vp
}

func (c *viewCtl) zoomH(steps int) {
	s := c.get()
	if s == nil || steps == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	center := c.x0 + c.span()/2
	c.ppb *= math.Pow(zoomStep, float64(steps))
	c.clamp(s)
	c.x0 = center - c.span()/2
	c.clamp(s)
	c.lastUser = time.Now()
}

func (c *viewCtl) zoomV(steps int) {
	s := c.get()
	if s == nil || steps == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	c.laneH *= math.Pow(zoomStep, float64(steps))
	c.clamp(s)
}

func (c *viewCtl) scrollT(ticks int) {
	s := c.get()
	if s == nil || ticks == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	c.x0 += float64(ticks) * jogPxPerTick / c.ppb
	c.clamp(s)
	c.lastUser = time.Now()
}

func (c *viewCtl) scrollPage(dir int) {
	s := c.get()
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	c.x0 += float64(dir) * c.span() / 4
	c.clamp(s)
	c.lastUser = time.Now()
}

func (c *viewCtl) scrollTracks(dir int) {
	s := c.get()
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	c.first += float64(dir)
	c.clamp(s)
}

// follow scrolls smoothly so the playhead stays at 25% of the clip area while
// playing (unless the user just moved the view by hand).
func (c *viewCtl) follow(t float64, playing bool) {
	s := c.get()
	if s == nil || !playing {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.lastUser) < followHold {
		return
	}
	c.clamp(s)
	c.x0 = t - c.span()*0.25
	c.clamp(s)
}

// reveal makes sure time t is on screen; recentres the view if not.
func (c *viewCtl) reveal(t float64) {
	s := c.get()
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	span := c.span()
	if t < c.x0+span*0.03 || t > c.x0+span*0.97 {
		c.x0 = t - span/2
		c.clamp(s)
	}
}

// decodeRel: relative two's complement (1 = +1, 127 = -1).
func decodeRel(v uint8) int {
	if v == 0 {
		return 0
	}
	if v < 64 {
		return int(v)
	}
	return int(v) - 128
}

// handleCC applies one control-surface CC. Returns true if the view changed.
func (c *viewCtl) handleCC(cc, val uint8) bool {
	switch cc {
	case ccDPadUp, ccDPadDown, ccDPadLeft, ccDPadRight:
		if val == 0 { // act on press only
			return false
		}
		switch cc {
		case ccDPadUp:
			c.scrollTracks(-1)
		case ccDPadDown:
			c.scrollTracks(+1)
		case ccDPadLeft:
			c.scrollPage(-1)
		case ccDPadRight:
			c.scrollPage(+1)
		}
	default:
		return false
	}
	return true
}

// setToast shows a short message on the view for d.
func (c *viewCtl) setToast(msg string, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.toast, c.toastUntil = msg, time.Now().Add(d)
}
