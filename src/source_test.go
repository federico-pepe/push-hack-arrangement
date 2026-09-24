package main

import (
	"encoding/json"
	"testing"
)

const snapJSON = `{"t":"snapshot","tracks":[
 {"name":"Drums","kind":"group","color":16711680,"group":-1,"clips":[]},
 {"name":"Kick","kind":"midi","color":65280,"group":0,"clips":[[0,8,"A",1122867],[8,16,"B",0],[5,5,"empty",1]]}],
 "locators":[{"time":4,"name":"Drop"}],"loop":{"on":true,"start":8,"length":16},"sig":[3,4],"tempo":124}`

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
	if s.Length != 16 || s.BeatsPerBar != 3 || !s.Loop.On || len(s.Locators) != 1 {
		t.Fatalf("set %+v", s)
	}
}

func TestStoreMessageThenSetWithPlayhead(t *testing.T) {
	st := &store{}
	st.putMessage("x", "y")
	if s, msg, hint := st.get(); s != nil || msg != "x" || hint != "y" {
		t.Fatal("message state wrong")
	}
	st.putSet(&Set{Length: 8})
	st.putPos(3.5, true)
	s, _, _ := st.get()
	if s == nil || s.Playhead != 3.5 {
		t.Fatalf("playhead %+v", s)
	}
}
