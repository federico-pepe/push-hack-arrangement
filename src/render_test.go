package main

import (
	"image/color"
	"testing"
)

func testSet() *Set {
	return &Set{
		Tracks: []Track{
			{Name: "A", Clips: []Clip{{Start: 0, End: 8, Color: 0xFF0000}}},
			{Name: "B", Clips: []Clip{{Start: 4, End: 20, Color: 0x00FF00}}},
		},
		Locators: []Locator{{Time: 8}},
		Loop:     Loop{On: true, Start: 4, Length: 8},
		Length:   20,
	}
}

func TestRenderSize(t *testing.T) {
	s := testSet()
	b := renderArrangement(s, fitView(s)).Bounds()
	if b.Dx() != 960 || b.Dy() != 160 {
		t.Fatalf("got %v", b)
	}
}

func TestRenderEmptySetDoesNotPanic(t *testing.T) {
	s := &Set{}
	renderArrangement(s, fitView(s))
	renderMessage("x", "y")
}

func TestClipPixelsUseClipColour(t *testing.T) {
	s := testSet()
	v := fitView(s)
	img := renderArrangement(s, v)
	// middle of track A's clip, first lane
	x, y := v.x(2), rulerH+3
	c := img.NRGBAAt(x, y)
	if c.R != 255 || c.G != 0 {
		t.Fatalf("pixel %v not red", c)
	}
}

func TestFitViewCoversSong(t *testing.T) {
	s := testSet()
	v := fitView(s)
	if v.x(s.Length) >= screenW {
		t.Fatal("song end off screen")
	}
}

func TestOverriddenDimsClips(t *testing.T) {
	s := testSet()
	v := fitView(s)
	crisp := renderArrangement(s, v).NRGBAAt(v.x(2), rulerH+3)
	s.Overridden = true
	dimmed := renderArrangement(s, v).NRGBAAt(v.x(2), rulerH+3)
	if crisp.R != 255 || dimmed.R >= 100 || dimmed.R == 0 {
		t.Fatalf("crisp %v dimmed %v", crisp, dimmed)
	}
}

func TestToastDraws(t *testing.T) {
	s := testSet()
	v := fitView(s)
	plain := renderArrangement(s, v)
	v.toastLines = []string{"ZOOM: TIME"}
	withToast := renderArrangement(s, v)
	same := true
	for i := range plain.Pix {
		if plain.Pix[i] != withToast.Pix[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("toast must change the image")
	}
}

func TestJoinToastLinesCombinesForOneStatusBarRow(t *testing.T) {
	if got := joinToastLines([]string{"A", "B"}); got != "A   B" {
		t.Fatalf("got %q", got)
	}
	if got := joinToastLines([]string{"solo"}); got != "solo" {
		t.Fatalf("got %q", got)
	}
}

func TestAnchorMarkerDraws(t *testing.T) {
	s := testSet()
	v := fitView(s)
	plain := renderArrangement(s, v)
	s.HasAnchor, s.Anchor = true, 2
	withAnchor := renderArrangement(s, v)
	x, y := v.x(2), rulerH+2
	if plain.NRGBAAt(x, y) == withAnchor.NRGBAAt(x, y) {
		t.Fatal("anchor marker must change the image")
	}
}

func TestCursorMarkerDraws(t *testing.T) {
	s := testSet()
	v := fitView(s)
	plain := renderArrangement(s, v)
	s.HasCursor, s.Cursor = true, 12
	withCursor := renderArrangement(s, v)
	x, y := v.x(12), screenH/2
	if plain.NRGBAAt(x, y) == withCursor.NRGBAAt(x, y) {
		t.Fatal("cursor marker must change the image")
	}
	if got := withCursor.NRGBAAt(x, y); got != colCursor {
		t.Fatalf("cursor pixel should be colCursor, got %v", got)
	}
}

func TestLoopBoundaryLinesOnlyWhenOn(t *testing.T) {
	s := testSet() // Loop: On=true, Start=4, Length=8 -> boundary at 4 and 12
	v := fitView(s)
	x0, x1 := v.x(4), v.x(12)
	on := renderArrangement(s, v)
	if on.NRGBAAt(x0, screenH/2) != colLoop || on.NRGBAAt(x1, screenH/2) != colLoop {
		t.Fatal("loop boundaries must be drawn (colLoop) when Loop.On")
	}
	s.Loop.On = false
	off := renderArrangement(s, v)
	if off.NRGBAAt(x0, screenH/2) == colLoop || off.NRGBAAt(x1, screenH/2) == colLoop {
		t.Fatal("loop boundaries must NOT be drawn when Loop.On is false")
	}
}

func TestLoopBraceStillShowsRegionWhenOff(t *testing.T) {
	s := testSet()
	s.Loop.On = false
	v := fitView(s)
	x := v.x(6) // inside the loop region [4,12)
	if renderArrangement(s, v).NRGBAAt(x, loopBraceY) != colLoop {
		t.Fatal("the horizontal brace should still mark the region even when Loop is off")
	}
}

func TestSelectedLocatorDrawsWhiteNotSky(t *testing.T) {
	s := testSet() // one locator at time=8
	v := fitView(s)
	x := v.x(8)
	unselected := renderArrangement(s, v).NRGBAAt(x, screenH/2)
	if unselected != colLoc {
		t.Fatalf("untouched locator should be colLoc, got %v", unselected)
	}
	s.HasSelLocator, s.SelLocator = true, 8
	selected := renderArrangement(s, v).NRGBAAt(x, screenH/2)
	if selected != colPlay {
		t.Fatalf("selected locator should be colPlay (white), got %v", selected)
	}
}

func TestTouchOverlayDimsAndDrawsText(t *testing.T) {
	s := testSet()
	plain := renderArrangement(s, fitView(s))
	vp := fitView(s)
	vp.overlay = []overlayCol{{col: 0, button: "Loop", toggle: true, buttonOn: true, name: "Loop Start", value: "1.1.1"}}
	over := renderArrangement(s, vp)
	// a pixel far from the text must be darker
	a, b := plain.NRGBAAt(900, 100), over.NRGBAAt(900, 100)
	if int(b.R)+int(b.G)+int(b.B) >= int(a.R)+int(a.G)+int(a.B) && a != (color.NRGBA{A: 255}) {
		t.Fatalf("not dimmed: %v -> %v", a, b)
	}
	// text drew something bright in column 0
	bright := false
	for y := 0; y < 60 && !bright; y++ {
		for x := 0; x < 120; x++ {
			if c := over.NRGBAAt(x, y); c.R > 150 || c.G > 150 {
				bright = true
				break
			}
		}
	}
	if !bright {
		t.Fatal("no overlay text found in column 0")
	}
}
