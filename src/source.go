package main

// Where the arrangement comes from: the PushHackArrangement Remote Script
// running inside Live. It streams the OPEN set (also unsaved edits) over a
// local Unix socket, one JSON per line. See remote-script/.

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
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
	bta      bool      // a track is off the arrangement (Back to Arrangement is lit)
	hold     time.Time // ignore Live's time until then (we just moved it ourselves)
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

// putPos stores what Live reported. The time is ignored for a short while after
// our own jog move, so a late report does not pull the playhead back.
func (st *store) putPos(t float64, playing, bta bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if time.Now().After(st.hold) {
		st.playhead = t
	}
	st.playing, st.bta = playing, bta
}

// setLocalTime moves the playhead at once (jog), before Live confirms.
func (st *store) setLocalTime(t float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.playhead = t
	st.hold = time.Now().Add(localHold)
}

const localHold = 400 * time.Millisecond

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
	Loop struct {
		On     bool    `json:"on"`
		Start  float64 `json:"start"`
		Length float64 `json:"length"`
	} `json:"loop"`
	Sig     [2]int  `json:"sig"`
	Tempo   float64 `json:"tempo"`
	Time    float64 `json:"time"`
	Playing bool    `json:"playing"`
	BTA     bool    `json:"bta"`
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
	s.Loop = Loop{On: m.Loop.On, Start: m.Loop.Start, Length: m.Loop.Length}
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
			st.putPos(m.Time, m.Playing, m.BTA)
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
