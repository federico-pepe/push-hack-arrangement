package main

import (
	"sync"
	"testing"
	"time"
)

func TestGridForZoom(t *testing.T) {
	if g := gridFor(1.3); g < 4 { // zoomed out: coarse
		t.Fatalf("grid %v too fine for fit view", g)
	}
	if g := gridFor(300); g != 1.0/16 { // zoomed in: 10px/300 = 0.033 -> 1/16 beat is the next value... 1/32 < want
		t.Fatalf("grid %v", g)
	}
	if g := gridFor(1e-6); g != 32 {
		t.Fatalf("cap %v", g)
	}
}

func TestJogTarget(t *testing.T) {
	cases := []struct {
		t, step float64
		ticks   int
		want    float64
	}{
		{4, 0.25, 1, 4.25},   // aligned
		{4, 0.25, -2, 3.5},   // aligned, back
		{4.1, 0.25, 1, 4.25}, // off grid: nearest grid line forward
		{4.1, 0.25, -1, 4.0}, // off grid: nearest grid line back
		{4.1, 0.25, 3, 4.75}, // off grid, 3 ticks
		{0.1, 0.25, -1, 0},   // clamp at 0
		{2, 0.25, 0, 2},
	}
	for _, c := range cases {
		if got := jogTarget(c.t, c.step, c.ticks); got < c.want-1e-9 || got > c.want+1e-9 {
			t.Errorf("jogTarget(%v,%v,%d)=%v want %v", c.t, c.step, c.ticks, got, c.want)
		}
	}
}

func TestJogMultiplier(t *testing.T) {
	if jogMultiplier(200*time.Millisecond) != 1 || jogMultiplier(50*time.Millisecond) != 2 ||
		jogMultiplier(30*time.Millisecond) != 4 || jogMultiplier(10*time.Millisecond) != 8 {
		t.Fatal("ramp wrong")
	}
}

type rig struct {
	c       *controls
	st      *store
	mu      sync.Mutex
	cmds    []map[string]any
	fb      []string
	on      bool
	shift   bool
	redraws int
}

func newRig() *rig {
	r := &rig{on: true}
	s := viewSet()
	r.st = &store{}
	r.st.putSet(s)
	vc := newViewCtl(func() *Set { x, _, _ := r.st.get(); return x })
	r.c = newControls(vc, r.st,
		func() bool { r.mu.Lock(); defer r.mu.Unlock(); return r.on },
		func() bool { r.mu.Lock(); defer r.mu.Unlock(); return r.shift },
		func() { r.mu.Lock(); r.redraws++; r.mu.Unlock() },
		func(m map[string]any) { r.mu.Lock(); r.cmds = append(r.cmds, m); r.mu.Unlock() },
		func(w string) { r.mu.Lock(); r.fb = append(r.fb, w); r.mu.Unlock() })
	return r
}

func (r *rig) sent() []map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]map[string]any(nil), r.cmds...)
}

func TestJogMovesPlayheadAndSendsSetTime(t *testing.T) {
	r := newRig()
	r.st.putPos(8, false, false)
	r.c.onCC(ccJog, 1)
	got := r.sent()
	if len(got) != 1 || got[0]["t"] != "set_time" {
		t.Fatalf("cmds %v", got)
	}
	if tt, _ := r.st.pos(); tt <= 8 {
		t.Fatalf("playhead did not move forward: %v", tt)
	}
}

func TestShiftJogScrollsViewNotPlayhead(t *testing.T) {
	r := newRig()
	r.shift = true
	r.c.vc.zoomH(30)
	before := r.c.vc.viewport(viewSet()).x0
	r.c.onCC(ccJog, 3)
	if len(r.sent()) != 0 {
		t.Fatal("shift+jog must not send set_time")
	}
	if r.c.vc.viewport(viewSet()).x0 <= before {
		t.Fatal("view did not scroll")
	}
}

func TestPlayAndSession(t *testing.T) {
	r := newRig()
	r.c.onCC(ccPlay, 127)
	r.c.onCC(ccPlay, 0) // release ignored
	r.c.onCC(ccSession, 127)
	r.shift = true
	r.c.onCC(ccSession, 127) // Shift+Session: chord, no bta
	got := r.sent()
	if len(got) != 2 || got[0]["t"] != "play_toggle" || got[1]["t"] != "bta" {
		t.Fatalf("cmds %v", got)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.fb) != 2 || r.fb[0] != "play" || r.fb[1] != "bta" {
		t.Fatalf("feedback %v", r.fb)
	}
}

func TestIgnoredWhenModeOff(t *testing.T) {
	r := newRig()
	r.on = false
	r.c.onCC(ccPlay, 127)
	r.c.onCC(ccJog, 1)
	if len(r.sent()) != 0 {
		t.Fatal("must ignore input while off")
	}
}

func TestDpadRepeatsWhileHeldAndStopsOnRelease(t *testing.T) {
	r := newRig()
	r.c.vc.zoomV(60) // tall lanes -> track scrolling possible
	r.c.onCC(ccDPadDown, 127)
	r.mu.Lock()
	first := r.redraws
	r.mu.Unlock()
	if first != 1 {
		t.Fatalf("press must act once at once, got %d", first)
	}
	time.Sleep(repeatDelay + 3*repeatEvery)
	r.mu.Lock()
	n := r.redraws
	r.mu.Unlock()
	if n < 3 {
		t.Fatalf("expected repeats while held, got %d", n)
	}
	r.c.onCC(ccDPadDown, 0)
	time.Sleep(3 * repeatEvery)
	r.mu.Lock()
	m := r.redraws
	r.mu.Unlock()
	time.Sleep(3 * repeatEvery)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.redraws != m {
		t.Fatal("repeat continued after release")
	}
}
