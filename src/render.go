package main

import (
	"image"
	"image/color"
	"strconv"

	"github.com/federico-pepe/ableton-push-hack/core/gfx"
	"github.com/federico-pepe/ableton-push-hack/core/gfx/text"
)

// Push 3 screen size in pixels.
const (
	screenW = 960
	screenH = 160
	rulerH  = 14
)

var (
	colBG     = color.NRGBA{0, 0, 0, 255}
	colGutter = color.NRGBA{22, 22, 22, 255}
	colGrid   = color.NRGBA{34, 34, 34, 255}
	colBar    = color.NRGBA{90, 90, 90, 255}
	colText   = color.NRGBA{220, 220, 220, 255}
	colPlay   = color.NRGBA{255, 255, 255, 255}
	colLoop   = color.NRGBA{255, 200, 0, 255}
	colLoc    = color.NRGBA{0, 200, 255, 255}
)

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255}
}

// viewport: what part of the arrangement is on screen.
type viewport struct {
	x0     float64 // time at left edge of the clip area (beats)
	ppb    float64 // pixels per beat
	laneH  float64 // pixels per track lane
	first  float64 // first visible track (fractional)
	gutter int     // width of the track-name column (0 = none)
}

// fitView shows the whole song, every track.
func fitView(s *Set) viewport {
	return viewport{ppb: fitPPB(s), laneH: fitLane(s)}
}

func (v viewport) x(beat float64) int { return v.gutter + int((beat-v.x0)*v.ppb) }

// fill draws a rectangle clipped to [minX,screenW) x [minY,screenH).
func fill(img *image.NRGBA, x, y, w, h, minX, minY int, c color.NRGBA) {
	if x < minX {
		w -= minX - x
		x = minX
	}
	if y < minY {
		h -= minY - y
		y = minY
	}
	if x+w > screenW {
		w = screenW - x
	}
	if y+h > screenH {
		h = screenH - y
	}
	if w > 0 && h > 0 {
		gfx.FillRect(img, x, y, w, h, c)
	}
}

// dimFactor: clip colours when Session clips override the arrangement.
const dimFactor = 0.35

func dim(c color.NRGBA) color.NRGBA {
	return color.NRGBA{uint8(float64(c.R) * dimFactor), uint8(float64(c.G) * dimFactor), uint8(float64(c.B) * dimFactor), 255}
}

func luma(c color.NRGBA) int { return (int(c.R)*299 + int(c.G)*587 + int(c.B)*114) / 1000 }

// renderArrangement draws the set. Text is ASCII only: the panel font has no other glyphs.
func renderArrangement(s *Set, v viewport) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, screenW, screenH))
	gfx.FillRect(img, 0, 0, screenW, screenH, colBG)

	drawRuler(img, v, s.bpb())

	top := rulerH + 1
	g := v.gutter
	for i, tr := range s.Tracks {
		y0 := top + int((float64(i)-v.first)*v.laneH)
		y1 := top + int((float64(i+1)-v.first)*v.laneH)
		if y1 <= top || y0 >= screenH {
			continue
		}
		h := y1 - y0
		if h > 2 {
			h-- // 1 px gap between lanes
		}
		for _, c := range tr.Clips {
			x0, x1 := v.x(c.Start), v.x(c.End)
			if x1 <= g || x0 >= screenW {
				continue
			}
			w := x1 - x0
			if w < 1 {
				w = 1
			}
			col := rgb(c.Color)
			if s.Overridden {
				col = dim(col)
			}
			fill(img, x0, y0, w, h, g, top, col)
			// clip name when there is room
			if v.laneH >= namesMinLane && w >= 36 && y0 >= top && y0+h <= screenH {
				tc := colBG
				if luma(col) < 110 {
					tc = colPlay
				}
				vis := x0
				if vis < g {
					vis = g
				}
				if x1-vis >= 36 {
					text.Draw(img, vis+3, y0+10, text.Truncate(c.Name, (x1-vis-6)/7), tc)
				}
			}
		}
		if g > 0 {
			fill(img, 0, y0, g, h, 0, top, colGutter)
			fill(img, 0, y0, 3, h, 0, top, rgb(tr.Color))
			if y0 >= top && y0+h <= screenH {
				ind := 0
				if tr.GroupID >= 0 {
					ind = 8
				}
				text.Draw(img, 7+ind, y0+10, text.Truncate(tr.Name, (g-10-ind)/7), colText)
			}
		}
	}

	// locators
	for _, l := range s.Locators {
		if x := v.x(l.Time); x >= v.gutter && x < screenW {
			gfx.FillRect(img, x, 0, 1, screenH, colLoc)
		}
	}
	// loop brace on the ruler
	if s.Loop.Length > 0 {
		x0, x1 := v.x(s.Loop.Start), v.x(s.Loop.Start+s.Loop.Length)
		if x1 > v.gutter && x0 < screenW {
			fill(img, x0, rulerH-2, x1-x0, 2, v.gutter, 0, colLoop)
		}
	}
	// playhead (saved insert marker)
	if x := v.x(s.Playhead); x >= v.gutter && x < screenW {
		fill(img, x, 0, 2, screenH, v.gutter, 0, colPlay)
	}

	return img
}

// drawRuler: bar numbers (labels >= 48 px apart), beat ticks when zoomed in.
func drawRuler(img *image.NRGBA, v viewport, bpb float64) {
	gfx.FillRect(img, v.gutter, rulerH, screenW-v.gutter, 1, colBar)
	barPx := v.ppb * bpb
	step := 1
	for float64(step)*barPx < 48 {
		step *= 2
	}
	first := int(v.x0 / bpb)
	if first < 0 {
		first = 0
	}
	if v.ppb >= 12 { // beat lines
		for b := int(v.x0); ; b++ {
			x := v.x(float64(b))
			if x >= screenW {
				break
			}
			if x >= v.gutter && float64(b) != float64(int(float64(b)/bpb))*bpb {
				fill(img, x, 6, 1, rulerH-6, v.gutter, 0, colBar)
			}
		}
	}
	for bar := first - first%step; ; bar += step {
		x := v.x(float64(bar) * bpb)
		if x >= screenW {
			break
		}
		if x < v.gutter {
			continue
		}
		fill(img, x, rulerH+1, 1, screenH-rulerH-1, v.gutter, 0, colGrid)
		fill(img, x, 0, 1, rulerH, v.gutter, 0, colBar)
		text.Draw(img, x+3, 11, strconv.Itoa(bar+1), colText)
	}
}

// renderMessage: full-screen text, for errors ("no set found").
func renderMessage(msg, hint string) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, screenW, screenH))
	gfx.FillRect(img, 0, 0, screenW, screenH, colBG)
	text.DrawScaled(img, 20, 60, 2, msg, colPlay)
	if hint != "" {
		text.Draw(img, 20, 90, hint, colText)
	}
	return img
}

func (s *Set) bpb() float64 {
	if s.BeatsPerBar > 0 {
		return s.BeatsPerBar
	}
	return 4
}
