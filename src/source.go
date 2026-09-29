package main

// Where the arrangement comes from: the PushHackArrangement Remote Script
// running inside Live. It streams the OPEN set (also unsaved edits) over a
// local Unix socket, one JSON per line. See remote-script/.

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"sort"
	"sync"
	"time"
)

const liveSocketPath = "/tmp/push-hack-arrangement.sock"

const msgNotConnected = "Remote Script not connected"

type store struct {
	mu       sync.Mutex
	set      *Set
	msg      string
	hint     string
	playhead float64
	playing  bool
	bta      bool // a track is off the arrangement (Back to Arrangement is lit)

	loopOn                bool
	loopStart, loopLength float64
	punchIn, punchOut     bool

	// hold: ignore Live's reported time/loop-start/length until this passes.
	// Set on a local jog or loop-knob edit, so the screen shows OUR value at
	// once instead of waiting a round trip to the Remote Script and back.
	// Booleans (playing, bta, loop on/off, punch) don't need this: a button
	// press is a one-shot action, so the ~100-250ms round trip is not felt
	// the way it would be while a knob is turning continuously.
	hold time.Time

	// Play-start anchor (jog wheel press). Local only: never sent by Live.
	hasAnchor bool
	anchor    float64

	// Locator last jumped to with Page </>. Local only, matched by time.
	hasSelLocator bool
	selLocator    float64

	// Scrub cursor (plain jog wheel). Local only, never sent to or echoed by
	// Live: this is the whole point -- looking around without seeking. A real
	// seek (setLocalTime, from Shift+jog, locator nav or the anchor jump)
	// drops it, since the real playhead is now wherever the cursor pointed to.
	hasCursor bool
	cursor    float64
}

// get returns a copy of the set with the live playhead, or a message to show.
func (st *store) get() (*Set, string, string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.set == nil {
		return nil, st.msg, st.hint
	}
	s := *st.set // slices shared, read-only
	s.Playhead = st.playhead
	s.Overridden = st.bta
	s.Loop = Loop{On: st.loopOn, Start: st.loopStart, Length: st.loopLength}
	s.PunchIn, s.PunchOut = st.punchIn, st.punchOut
	s.HasAnchor, s.Anchor = st.hasAnchor, st.anchor
	s.HasSelLocator, s.SelLocator = st.hasSelLocator, st.selLocator
	s.HasCursor, s.Cursor = st.hasCursor, st.cursor
	return &s, "", ""
}

func (st *store) putSet(s *Set) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.set, st.msg, st.hint = s, "", ""
}

func (st *store) putMessage(msg, hint string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.set, st.msg, st.hint = nil, msg, hint
}

func (st *store) pos() (float64, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.playhead, st.playing
}

func (st *store) state() (playing, bta bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.playing, st.bta
}

// transport returns the loop and punch state (loop on/off, punch in/out, loop
// start and length in beats).
func (st *store) transport() (loopOn, punchIn, punchOut bool, loopStart, loopLength float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.loopOn, st.punchIn, st.punchOut, st.loopStart, st.loopLength
}

// putPos stores what Live reported. Time and loop start/length are ignored for
// a short while after our own edit, so a late report does not pull them back;
// the on/off flags always update at once (see the `hold` field's comment).
func (st *store) putPos(t float64, playing, bta, loopOn bool, loopStart, loopLength float64, punchIn, punchOut bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if time.Now().After(st.hold) {
		st.playhead = t
		st.loopStart, st.loopLength = loopStart, loopLength
	}
	if bta != st.bta {
		log.Printf("back to arrangement available: %v", bta)
	}
	st.playing, st.bta = playing, bta
	st.loopOn, st.punchIn, st.punchOut = loopOn, punchIn, punchOut
}

// setLocalTime moves the playhead at once (jog), before Live confirms.
func (st *store) setLocalTime(t float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.playhead = t
	st.hold = time.Now().Add(localHold)
	st.hasCursor = false // a real seek: the cursor's job is done
}

// setCursor / clearCursor / getCursor: the scrub cursor. Local only, and
// unlike setLocalTime, NOT reconciled against anything Live reports -- there
// is nothing to reconcile, since we never told Live to move.
func (st *store) setCursor(t float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.hasCursor, st.cursor = true, t
}

func (st *store) clearCursor() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.hasCursor = false
}

func (st *store) getCursor() (float64, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.cursor, st.hasCursor
}

// setLocalLoop moves the loop start/length at once (the loop knobs), before
// Live confirms.
func (st *store) setLocalLoop(start, length float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.loopStart, st.loopLength = start, length
	st.hold = time.Now().Add(localHold)
}

const localHold = 400 * time.Millisecond

// setAnchor / clearAnchor / getAnchor: the play-start anchor (jog wheel
// press). Local only -- Live never reports it back.
func (st *store) setAnchor(t float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.hasAnchor, st.anchor = true, t
}

func (st *store) clearAnchor() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.hasAnchor = false
}

func (st *store) getAnchor() (float64, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.anchor, st.hasAnchor
}

// setSelLocator remembers which locator (by time) Page </> last jumped to,
// so the render can draw its line white instead of light blue.
func (st *store) setSelLocator(t float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.hasSelLocator, st.selLocator = true, t
}

const hintActivate = "Enable PushHackArrangement in Live Preferences (control surface, Input/Output None), restart Live"

// wire format ---------------------------------------------------------------

type wireClip struct {
	Start, End float64
	Name       string
	Color      uint32
}

func (c *wireClip) UnmarshalJSON(b []byte) error {
	var a []json.RawMessage
	if err := json.Unmarshal(b, &a); err != nil || len(a) < 4 {
		return err
	}
	_ = json.Unmarshal(a[0], &c.Start)
	_ = json.Unmarshal(a[1], &c.End)
	_ = json.Unmarshal(a[2], &c.Name)
	_ = json.Unmarshal(a[3], &c.Color)
	return nil
}

type wireMsg struct {
	T      string `json:"t"`
	Tracks []struct {
		Name  string     `json:"name"`
		Kind  string     `json:"kind"`
		Color uint32     `json:"color"`
		Group int        `json:"group"`
		Clips []wireClip `json:"clips"`
	} `json:"tracks"`
	Locators []struct {
		Time float64 `json:"time"`
		Name string  `json:"name"`
	} `json:"locators"`
	Sig   [2]int  `json:"sig"`
	Tempo float64 `json:"tempo"`

	// "pos" fields. Loop/punch live here, not in the snapshot: "pos" is sent
	// every tick it changes, so a toggle or a knob turn is reflected almost
	// at once, instead of waiting for the next (much rarer) snapshot.
	Time       float64 `json:"time"`
	Playing    bool    `json:"playing"`
	BTA        bool    `json:"bta"`
	LoopOn     bool    `json:"loop_on"`
	LoopStart  float64 `json:"loop_start"`
	LoopLength float64 `json:"loop_length"`
	PunchIn    bool    `json:"punch_in"`
	PunchOut   bool    `json:"punch_out"`
}

func setFromSnapshot(m *wireMsg) *Set {
	s := &Set{Name: "Live Set", BeatsPerBar: 4}
	if m.Sig[0] > 0 && m.Sig[1] > 0 {
		s.BeatsPerBar = float64(m.Sig[0]) * 4 / float64(m.Sig[1])
	}
	for i, t := range m.Tracks {
		tr := Track{Name: t.Name, Kind: t.Kind, Color: t.Color, GroupID: t.Group, ID: i}
		for _, c := range t.Clips {
			if c.End <= c.Start {
				continue
			}
			col := c.Color
			if col == 0 {
				col = t.Color
			}
			tr.Clips = append(tr.Clips, Clip{Start: c.Start, End: c.End, Name: c.Name, Color: col})
			if c.End > s.Length {
				s.Length = c.End
			}
		}
		s.Tracks = append(s.Tracks, tr)
	}
	for _, l := range m.Locators {
		s.Locators = append(s.Locators, Locator{Time: l.Time, Name: l.Name})
	}
	sort.Slice(s.Locators, func(i, j int) bool { return s.Locators[i].Time < s.Locators[j].Time })
	// Loop/punch are not in the snapshot (see wireMsg): the store overlays
	// them from the last "pos" message, the same way it overlays Playhead.
	return s
}

// runLiveSource keeps a connection to the Remote Script; reconnects on loss.
// onChange is called after every update that changes what is drawn.
func runLiveSource(st *store, onChange func(), stop <-chan struct{}) {
	st.putMessage(msgNotConnected, hintActivate)
	for {
		conn, err := net.DialTimeout("unix", liveSocketPath, 2*time.Second)
		if err == nil {
			link.set(conn)
			log.Print("Remote Script connected")
			st.putMessage("Waiting for Live Set data", "")
			readLive(conn, st, onChange, stop)
			link.set(nil)
			conn.Close()
			log.Print("Remote Script disconnected")
			st.putMessage(msgNotConnected, hintActivate)
			onChange()
		}
		select {
		case <-stop:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func readLive(conn net.Conn, st *store, onChange func(), stop <-chan struct{}) {
	go func() { <-stop; conn.Close() }()
	r := bufio.NewReaderSize(conn, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		var m wireMsg
		if err := json.Unmarshal(line, &m); err != nil {
			log.Printf("bad message: %v", err)
			continue
		}
		switch m.T {
		case "snapshot":
			s := setFromSnapshot(&m)
			st.putSet(s)
			log.Printf("snapshot: %d tracks, %.0f beats, %d locators", len(s.Tracks), s.Length, len(s.Locators))
			onChange()
		case "pos":
			st.putPos(m.Time, m.Playing, m.BTA, m.LoopOn, m.LoopStart, m.LoopLength, m.PunchIn, m.PunchOut)
			onChange()
		}
	}
}

// link: the current connection, for commands to Live.
type liveLink struct {
	mu   sync.Mutex
	conn net.Conn
}

var link liveLink

func (l *liveLink) set(c net.Conn) {
	l.mu.Lock()
	l.conn = c
	l.mu.Unlock()
}

// sendCmd sends one command line to the Remote Script. Dropped if not connected.
func sendCmd(m map[string]any) {
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	link.mu.Lock()
	defer link.mu.Unlock()
	if link.conn == nil {
		return
	}
	_ = link.conn.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
	_, _ = link.conn.Write(append(b, '\n'))
}
