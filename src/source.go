package main

// Holds the current Set. Source of data = newest saved .als under setsRoot,
// reloaded when its mtime changes. (Remote Script source comes later.)

import (
	"log"
	"os"
	"sync"
	"time"
)

const setsRoot = "/data/Music/Ableton/Sets"

type store struct {
	mu  sync.Mutex
	set *Set
	err string
}

func (st *store) get() (*Set, string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.set, st.err
}

func (st *store) put(s *Set, err string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.set, st.err = s, err
}

// watchSets loads at once, then polls. onChange runs after a new Set is stored.
func watchSets(st *store, fixed string, onChange func(), stop <-chan struct{}) {
	var lastPath string
	var lastMod time.Time
	load := func() {
		path := fixed
		if path == "" {
			p, err := FindNewestSet(setsRoot)
			if err != nil {
				if cur, _ := st.get(); cur == nil {
					st.put(nil, "No Live Set found")
				}
				return
			}
			path = p
		}
		info, err := os.Stat(path)
		if err != nil || (path == lastPath && info.ModTime().Equal(lastMod)) {
			return
		}
		s, err := LoadSetFile(path)
		if err != nil {
			log.Printf("load %s: %v", path, err)
			if cur, _ := st.get(); cur == nil {
				st.put(nil, "Cannot read Live Set")
			}
			return
		}
		lastPath, lastMod = path, info.ModTime()
		st.put(s, "")
		log.Printf("loaded %q: %d tracks, %.0f beats, %d locators", s.Name, len(s.Tracks), s.Length, len(s.Locators))
		onChange()
	}
	load()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			load()
		}
	}
}
