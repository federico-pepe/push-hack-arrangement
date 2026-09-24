package main

import (
	"image"
	"image/color"

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
	colBG    = color.NRGBA{0, 0, 0, 255}
	colGrid  = color.NRGBA{40, 40, 40, 255}
	colBar   = color.NRGBA{90, 90, 90, 255}
	colText  = color.NRGBA{220, 220, 220, 255}
	colPlay  = color.NRGBA{255, 255, 255, 255}
	testClip = []color.NRGBA{
		{230, 80, 80, 255}, {240, 170, 60, 255}, {90, 200, 110, 255}, {80, 150, 240, 255},
	}
)

// renderTestFrame draws a fixed pattern (ruler, 4 lanes, clips, playhead) to
// prove the display path. Text is ASCII only: the panel font has no other glyphs.
func renderTestFrame() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, screenW, screenH))
	gfx.FillRect(img, 0, 0, screenW, screenH, colBG)

	// ruler: one tick per "bar" (60 px), numbered
	for i, x := 0, 0; x < screenW; i, x = i+1, x+60 {
		gfx.FillRect(img, x, 0, 1, screenH, colGrid)
		gfx.FillRect(img, x, 0, 1, rulerH, colBar)
		text.Draw(img, x+3, 11, itoa(i+1), colText)
	}
	gfx.FillRect(img, 0, rulerH, screenW, 1, colBar)

	// 4 lanes with a few clips each
	laneH := (screenH - rulerH - 1) / 4
	for l := 0; l < 4; l++ {
		y := rulerH + 1 + l*laneH
		gfx.FillRect(img, 0, y+laneH-1, screenW, 1, colGrid)
		c := testClip[l]
		for k := 0; k < 5; k++ {
			x := 20 + k*170 + l*25
			gfx.FillRect(img, x, y+2, 130, laneH-5, c)
			text.Draw(img, x+4, y+14, "Clip "+itoa(k+1), colBG)
		}
	}

	gfx.FillRect(img, 300, rulerH, 2, screenH-rulerH, colPlay) // playhead
	text.Draw(img, screenW-150, screenH-6, "ARRANGEMENT TEST", colText)
	return img
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
