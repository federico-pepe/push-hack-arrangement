package main

import (
	"testing"
	"time"
)

func viewSet() *Set {
	s := &Set{Length: 200}
	for i := 0; i < 30; i++ {
		s.Tracks = append(s.Tracks, Track{Name: "T", GroupID: -1,
			Clips: []Clip{{Start: 0, End: 200, Color: 0x5480E4, Name: "clip"}}})
	}
	s.Locators = []Locator{{Time: 40, Name: "A"}, {Time: 100, Name: "B"}}
	return s
}

func newTestView() (*viewCtl, *Set) {
	s := viewSet()
	vc := newViewCtl(func() *Set { return s })
	vc.viewport(s)
	return vc, s
}

// decodeRel itself is core/push3.DecodeRel's own function (tested there,
// core/push3/encoder_test.go) -- this package only calls it.

func TestZoomHClampsAndKeepsCentre(t *testing.T) {
	vc, s := newTestView()
	vc.zoomH(-1000) // cannot go below fit
	if v := vc.viewport(s); v.ppb < fitPPB(s)-1e-9 {
		t.Fatalf("ppb below fit: %v", v.ppb)
	}
	vc.zoomH(1000) // capped
	if v := vc.viewport(s); v.ppb > maxPPB+1e-9 {
		t.Fatalf("ppb above max: %v", v.ppb)
	}
	// zoom pivots on the playhead: it stays at screen centre at any zoom level
	s.Playhead = 90
	vc = newViewCtl(func() *Set { return s })
	checkCentred := func() {
		v := vc.viewport(s)
		center := v.x0 + vc.span()/2
		if d := center - s.Playhead; d > 0.5 || d < -0.5 {
			t.Fatalf("playhead not centred: view centre %v playhead %v", center, s.Playhead)
		}
	}
	vc.zoomH(20)
	checkCentred()
	vc.zoomH(5)
	checkCentred()
}

func TestZoomHPivotsOnCursorWhenActive(t *testing.T) {
	s := viewSet()
	s.Playhead = 10
	s.HasCursor, s.Cursor = true, 90
	vc := newViewCtl(func() *Set { return s })
	vc.zoomH(20)
	v := vc.viewport(s)
	center := v.x0 + vc.span()/2
	if d := center - s.Cursor; d > 0.5 || d < -0.5 {
		t.Fatalf("must centre on the cursor (90), view centre is at %v", center)
	}
}

func TestZoomVAndTrackScroll(t *testing.T) {
	vc, s := newTestView()
	v0 := vc.viewport(s)
	if v0.gutter != 0 {
		t.Fatal("30 tracks fit -> no names")
	}
	vc.zoomV(40)
	v := vc.viewport(s)
	if v.laneH <= v0.laneH || v.gutter != gutterW {
		t.Fatalf("zoomed lanes should be taller with names: %+v", v)
	}
	vc.scrollTracks(-5)
	if vc.viewport(s).first != 0 {
		t.Fatal("first must clamp at 0")
	}
	vc.scrollTracks(1000)
	f := vc.viewport(s).first
	if max := float64(len(s.Tracks)) - laneAvail()/v.laneH; f > max+1e-9 {
		t.Fatalf("first %v beyond %v", f, max)
	}
}

func TestScrollAvailabilityAtEdges(t *testing.T) {
	vc, s := newTestView()
	up, down, left, right := vc.availability(s)
	if up || down || left || right {
		t.Fatalf("fit view shows everything: nothing should be scrollable, got %v %v %v %v", up, down, left, right)
	}

	vc.zoomH(30) // zoom in so there's room to scroll in time
	up, down, left, right = vc.availability(s)
	if left {
		t.Fatal("still at the start: left must be unavailable")
	}
	if !right {
		t.Fatal("zoomed in from the start: right must be available")
	}
	vc.scrollT(10000)
	_, _, _, right = vc.availability(s)
	if right {
		t.Fatal("scrolled to the end: right must now be unavailable")
	}

	vc.zoomV(60) // tall lanes -> room to scroll tracks
	_, down, _, _ = vc.availability(s)
	if !down {
		t.Fatal("zoomed in from the top: down must be available")
	}
	vc.scrollTracks(1000)
	_, down, _, _ = vc.availability(s)
	if down {
		t.Fatal("scrolled to the last track: down must now be unavailable")
	}
}

func TestScrollAvailabilityCallbackFiresOnMutators(t *testing.T) {
	vc, s := newTestView()
	var calls int
	var lastUp, lastDown, lastLeft, lastRight bool
	vc.onScrollAvailability(func(up, down, left, right bool) {
		calls++
		lastUp, lastDown, lastLeft, lastRight = up, down, left, right
	})
	vc.zoomH(1)
	if calls != 1 {
		t.Fatalf("zoomH must fire the callback once, got %d", calls)
	}
	vc.scrollT(1)
	vc.scrollPage(1)
	vc.scrollTracks(1)
	vc.zoomV(1)
	if calls != 5 {
		t.Fatalf("every mutator must fire it, got %d calls", calls)
	}
	wantUp, wantDown, wantLeft, wantRight := vc.availability(s)
	if lastUp != wantUp || lastDown != wantDown || lastLeft != wantLeft || lastRight != wantRight {
		t.Fatal("callback's last values must match the current availability")
	}
}

func TestScrollTimeClamps(t *testing.T) {
	vc, s := newTestView()
	vc.zoomH(30)
	vc.scrollT(-100000)
	if vc.viewport(s).x0 != 0 {
		t.Fatal("x0 must clamp at 0")
	}
	vc.scrollT(100000)
	v := vc.viewport(s)
	if v.x0+vc.span() > totalBeats(s)+1e-6 {
		t.Fatalf("scrolled past the end: %+v", v)
	}
}

func TestFollow(t *testing.T) {
	vc, s := newTestView()
	vc.zoomH(30)
	vc.scrollT(-100000)
	vc.follow(100, false)
	if vc.viewport(s).x0 != 0 {
		t.Fatal("no follow when stopped")
	}
	vc.lastUser = time.Now().Add(-time.Hour)
	vc.follow(100, true)
	v := vc.viewport(s)
	if 100 < v.x0 || 100 > v.x0+vc.span() {
		t.Fatalf("playhead not in view: %+v", v)
	}
	// just moved by hand: do not fight the user
	vc.scrollT(-100000)
	vc.follow(150, true)
	if vc.viewport(s).x0 != 0 {
		t.Fatal("must not follow right after a manual move")
	}
}

func TestHandleCC(t *testing.T) {
	vc, _ := newTestView()
	if vc.handleCC(ccVolumeDial, 1) || vc.handleCC(ccTempoDial, 127) {
		t.Fatal("dials belong to controls, not the view")
	}
	if vc.handleCC(ccDPadUp, 0) {
		t.Fatal("dpad release must be ignored")
	}
	if !vc.handleCC(ccDPadDown, 127) {
		t.Fatal("dpad press handled")
	}
	if vc.handleCC(ccJog, 3) {
		t.Fatal("jog belongs to controls, not the view")
	}
	if vc.handleCC(99, 1) {
		t.Fatal("unknown cc must not be handled")
	}
}

func TestRenderZoomedWithNames(t *testing.T) {
	vc, s := newTestView()
	vc.zoomV(60)
	vc.zoomH(40)
	vc.scrollTracks(3)
	img := renderArrangement(s, vc.viewport(s))
	if img.Bounds().Dx() != 960 || img.Bounds().Dy() != 160 {
		t.Fatal("bad size")
	}
}
