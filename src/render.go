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

const beatsPerBar = 4 // TODO: read the real time signature

var (
	colBG    = color.NRGBA{0, 0, 0, 255}
	colGrid  = color.NRGBA{34, 34, 34, 255}
	colBar   = color.NRGBA{90, 90, 90, 255}
	colText  = color.NRGBA{220, 220, 220, 255}
	colPlay  = color.NRGBA{255, 255, 255, 255}
	colLoop  = color.NRGBA{255, 200, 0, 255}
	colLoc   = color.NRGBA{0, 200, 255, 255}
	colTitle = color.NRGBA{30, 30, 30, 255}
)

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255}
}

// viewport: what part of the arrangement is on screen.
type viewport struct {
	x0  float64 // time at left edge (beats)
	ppb float64 // pixels per beat
}

// fitView shows the whole song.
func fitView(s *Set) viewport {
	length := s.Length + 4
	if length < 16 {
		length = 16
	}
	return viewport{x0: 0, ppb: float64(screenW) / length}
}

func (v viewport) x(beat float64) int { return int((beat - v.x0) * v.ppb) }

// renderArrangement draws the set. Text is ASCII only: the panel font has no other glyphs.
func renderArrangement(s *Set, v viewport) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, screenW, screenH))
	gfx.FillRect(img, 0, 0, screenW, screenH, colBG)

	drawRuler(img, v)

	n := len(s.Tracks)
	top := rulerH + 1
	avail := screenH - top
	if n > 0 {
		for i, tr := range s.Tracks {
			y0 := top + i*avail/n
			y1 := top + (i+1)*avail/n
			h := y1 - y0
			if h > 2 {
				h-- // 1 px gap between lanes
			}
			for _, c := range tr.Clips {
				x0, x1 := v.x(c.Start), v.x(c.End)
				if x1 < 0 || x0 >= screenW {
					continue
				}
				if x0 < 0 {
					x0 = 0
				}
				w := x1 - x0
				if w < 1 {
					w = 1
				}
				if x0+w > screenW {
					w = screenW - x0
				}
				gfx.FillRect(img, x0, y0, w, h, rgb(c.Color))
			}
		}
	}

	// locators
	for _, l := range s.Locators {
		if x := v.x(l.Time); x >= 0 && x < screenW {
			gfx.FillRect(img, x, 0, 1, screenH, colLoc)
		}
	}
	// loop brace on the ruler
	if s.Loop.Length > 0 {
		x0, x1 := v.x(s.Loop.Start), v.x(s.Loop.Start+s.Loop.Length)
		if x1 > 0 && x0 < screenW {
			if x0 < 0 {
				x0 = 0
			}
			gfx.FillRect(img, x0, rulerH-2, x1-x0, 2, colLoop)
		}
	}
	// playhead (saved insert marker)
	if x := v.x(s.Playhead); x >= 0 && x < screenW {
		gfx.FillRect(img, x, 0, 2, screenH, colPlay)
	}

	// title, top right
	const title = "Arrangement Mode"
	tw := text.Width(title) + 10
	gfx.FillRect(img, screenW-tw, 0, tw, rulerH-1, colTitle)
	text.Draw(img, screenW-tw+5, 11, title, colPlay)
	return img
}

// drawRuler: bar numbers, spacing grows so labels stay >= 48 px apart.
func drawRuler(img *image.NRGBA, v viewport) {
	gfx.FillRect(img, 0, rulerH, screenW, 1, colBar)
	barPx := v.ppb * beatsPerBar
	step := 1
	for float64(step)*barPx < 48 {
		step *= 2
	}
	first := int(v.x0 / beatsPerBar)
	if first < 0 {
		first = 0
	}
	for bar := first - first%step; ; bar += step {
		x := v.x(float64(bar * beatsPerBar))
		if x >= screenW {
			break
		}
		if x < 0 {
			continue
		}
		gfx.FillRect(img, x, rulerH+1, 1, screenH-rulerH-1, colGrid)
		gfx.FillRect(img, x, 0, 1, rulerH, colBar)
		text.Draw(img, x+3, 11, strconv.Itoa(bar+1), colText)
	}
}

// renderMessage: full-screen text, for errors ("no set found").
func renderMessage(msg string) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, screenW, screenH))
	gfx.FillRect(img, 0, 0, screenW, screenH, colBG)
	text.DrawScaled(img, 20, 60, 2, "Arrangement Mode", colPlay)
	text.Draw(img, 20, 90, msg, colText)
	return img
}
