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

	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

// CCs, named from core/push3 rather than restated as magic numbers.
const (
	ccTempoDial  = push3.CCTempo
	ccTempoPress = push3.CCTempoPress
	ccVolumeDial = push3.CCVolume
	ccJog        = push3.CCJogWheel
	ccDPadLeft   = push3.CCDPadLeft
	ccDPadRight  = push3.CCDPadRight
	ccDPadUp     = push3.CCDPadUp
	ccDPadDown   = push3.CCDPadDown
	ccDPadCenter = push3.CCDPadCenter

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
		vp.toastLines = []string{c.toast}
	}
	return vp
}

// viewInfo: the 1-based first and last bar in view, and the number of tracks in view.
func (c *viewCtl) viewInfo(s *Set, bpb float64) (first, last, tracks int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	first = int(c.x0/bpb) + 1
	last = int((c.x0+c.span())/bpb) + 1
	tracks = int(laneAvail()/c.laneH + 0.5)
	if tracks > len(s.Tracks) {
		tracks = len(s.Tracks)
	}
	return
}

// zoomPivot: the horizontal zoom always keeps this point centred on screen --
// the scrub cursor while one is active (that's what the user is looking at),
// else the real playhead.
func zoomPivot(s *Set) float64 {
	if s.HasCursor {
		return s.Cursor
	}
	return s.Playhead
}

func (c *viewCtl) zoomH(steps int) {
	s := c.get()
	if s == nil || steps == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	center := zoomPivot(s)
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

// revealTrack scrolls the tracks the least needed so track i is fully in view.
func (c *viewCtl) revealTrack(i int) {
	s := c.get()
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clamp(s)
	visible := math.Floor(laneAvail() / c.laneH)
	if float64(i) < c.first {
		c.first = float64(i)
	} else if float64(i+1) > c.first+visible {
		c.first = float64(i+1) - visible
	}
	c.clamp(s)
}

// resetToFit: Delete + Tempo press. Shows the whole song, every track, exactly
// like the view starts out.
func (c *viewCtl) resetToFit() {
	s := c.get()
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ppb = fitPPB(s)
	c.laneH = fitLane(s)
	c.x0 = 0
	c.first = 0
	c.clamp(s)
	c.lastUser = time.Now()
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

// setToast shows a short message on the view for d.
func (c *viewCtl) setToast(msg string, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.toast, c.toastUntil = msg, time.Now().Add(d)
}
