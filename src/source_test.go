package main

import (
	"encoding/json"
	"testing"
	"time"
)

const snapJSON = `{"t":"snapshot","tracks":[
 {"name":"Drums","kind":"group","color":16711680,"group":-1,"clips":[]},
 {"name":"Kick","kind":"midi","color":65280,"group":0,"clips":[[0,8,"A",1122867],[8,16,"B",0],[5,5,"empty",1]]}],
 "locators":[{"time":9,"name":"B"},{"time":4,"name":"Drop"}],"sig":[3,4],"tempo":124}`

func TestSetFromSnapshot(t *testing.T) {
	var m wireMsg
	if err := json.Unmarshal([]byte(snapJSON), &m); err != nil {
		t.Fatal(err)
	}
	s := setFromSnapshot(&m)
	if len(s.Tracks) != 2 || s.Tracks[1].GroupID != 0 || s.Tracks[1].Kind != "midi" {
		t.Fatalf("tracks %+v", s.Tracks)
	}
	cl := s.Tracks[1].Clips
	if len(cl) != 2 { // zero-length clip dropped
		t.Fatalf("clips %+v", cl)
	}
	if cl[0].Color != 1122867 || cl[1].Color != 65280 { // 0 -> track colour
		t.Fatalf("colours %+v", cl)
	}
	if s.Length != 16 || s.BeatsPerBar != 3 {
		t.Fatalf("set %+v", s)
	}
	// snapshot carries locators out of order; setFromSnapshot must sort them.
	if len(s.Locators) != 2 || s.Locators[0].Time != 4 || s.Locators[1].Time != 9 {
		t.Fatalf("locators not sorted: %+v", s.Locators)
	}
	// Loop/punch are not in the snapshot -- they come from "pos" (see store).
	if s.Loop.On || s.Loop.Length != 0 {
		t.Fatalf("snapshot must not carry loop state: %+v", s.Loop)
	}
}

func TestStoreMessageThenSetWithPlayhead(t *testing.T) {
	st := &store{}
	st.putMessage("x", "y")
	if s, msg, hint := st.get(); s != nil || msg != "x" || hint != "y" {
		t.Fatal("message state wrong")
	}
	st.putSet(&Set{Length: 8})
	st.putPos(3.5, true, false, false, 0, 4, false, false)
	s, _, _ := st.get()
	if s == nil || s.Playhead != 3.5 {
		t.Fatalf("playhead %+v", s)
	}
}

func TestLocalTimeHoldsAgainstLatePos(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.setLocalTime(6)
	st.putPos(1, true, true, false, 0, 4, false, false) // late report from Live
	if tt, playing := st.pos(); tt != 6 || !playing {
		t.Fatalf("t=%v playing=%v", tt, playing)
	}
	if _, bta := st.state(); !bta {
		t.Fatal("bta must still update (booleans are not held)")
	}
}

func TestLocalLoopHoldsAgainstLatePos(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.setLocalLoop(16, 32)
	st.putPos(1, false, false, true, 0, 4, false, false) // late report: stale start/length
	on, _, _, start, length := st.transport()
	if start != 16 || length != 32 {
		t.Fatalf("local loop overwritten: start=%v length=%v", start, length)
	}
	if !on {
		t.Fatal("loop-on is a boolean: must update at once")
	}
}

// Regression: hold and loopHold used to be the same field, so adjusting the
// loop start/length (a knob turn keeps re-arming its hold) also suppressed
// every Live-reported playhead update -- the on-screen playhead would sit
// frozen for as long as you kept turning the knob, even while the song was
// playing. They must be independent.
func TestAdjustingLoopDoesNotFreezeThePlayhead(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.setLocalLoop(16, 32) // arms loopHold; must NOT touch the playhead's hold
	st.putPos(42, true, false, true, 0, 4, false, false)
	if tt, playing := st.pos(); tt != 42 || !playing {
		t.Fatalf("playhead must keep advancing while the loop is held: t=%v playing=%v", tt, playing)
	}
	_, _, _, start, length := st.transport()
	if start != 16 || length != 32 {
		t.Fatalf("loop should still be held: start=%v length=%v", start, length)
	}
}

// The mirror image: jogging/seeking must not freeze the loop's on-screen
// start/length either.
func TestSeekingDoesNotFreezeTheLoop(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.setLocalTime(6) // arms hold; must NOT touch the loop's hold
	st.putPos(1, false, false, true, 99, 111, false, false)
	_, _, _, start, length := st.transport()
	if start != 99 || length != 111 {
		t.Fatalf("loop must keep updating while the time is held: start=%v length=%v", start, length)
	}
	if tt, _ := st.pos(); tt != 6 {
		t.Fatalf("the time hold should still protect the playhead: %v", tt)
	}
}

func TestHoldExpiresAndLiveWins(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.setLocalTime(6)
	time.Sleep(localHold + 20*time.Millisecond)
	st.putPos(9, false, false, false, 0, 4, false, false)
	if tt, _ := st.pos(); tt != 9 {
		t.Fatalf("t=%v, want Live's report once the hold expires", tt)
	}
}

func TestStoreGetMarksOverridden(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.putPos(1, false, true, false, 0, 4, false, false)
	if s, _, _ := st.get(); !s.Overridden {
		t.Fatal("bta must mark the set overridden")
	}
	st.putPos(1, false, false, false, 0, 4, false, false)
	if s, _, _ := st.get(); s.Overridden {
		t.Fatal("no bta: crisp")
	}
}

func TestTransportAndPunch(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.putPos(1, false, false, true, 8, 16, true, false)
	on, in, out, start, length := st.transport()
	if !on || !in || out || start != 8 || length != 16 {
		t.Fatalf("transport %v %v %v %v %v", on, in, out, start, length)
	}
	s, _, _ := st.get()
	if !s.Loop.On || s.Loop.Start != 8 || s.Loop.Length != 16 || !s.PunchIn || s.PunchOut {
		t.Fatalf("set %+v", s)
	}
}

func TestAnchorSetClearGet(t *testing.T) {
	st := &store{}
	if _, ok := st.getAnchor(); ok {
		t.Fatal("no anchor by default")
	}
	st.setAnchor(12.5)
	if t2, ok := st.getAnchor(); !ok || t2 != 12.5 {
		t.Fatalf("anchor %v %v", t2, ok)
	}
	st.putSet(&Set{Length: 8})
	if s, _, _ := st.get(); !s.HasAnchor || s.Anchor != 12.5 {
		t.Fatalf("set does not carry the anchor: %+v", s)
	}
	st.clearAnchor()
	if _, ok := st.getAnchor(); ok {
		t.Fatal("anchor must be gone after clear")
	}
}

func TestSelLocator(t *testing.T) {
	st := &store{}
	st.putSet(&Set{Length: 8})
	st.setSelLocator(4)
	s, _, _ := st.get()
	if !s.HasSelLocator || s.SelLocator != 4 {
		t.Fatalf("set %+v", s)
	}
}
