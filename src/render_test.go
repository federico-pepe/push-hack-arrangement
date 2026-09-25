package main

import "testing"

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
	v.toast = "ZOOM: TIME"
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
