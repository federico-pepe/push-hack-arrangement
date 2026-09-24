package main

// Reader for a saved Live Set (.als = gzip XML). Streams the XML: sets are
// ~10 MB unpacked and Push has little RAM to spare.
//
// Only arrangement data is read: clips under ArrangerAutomation/Events.
// Session clips (ClipSlotList) are ignored.

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func LoadSetFile(path string) (*Set, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	s, err := parseSet(zr)
	if err != nil {
		return nil, err
	}
	s.Name = strings.TrimSuffix(filepath.Base(path), ".als")
	return s, nil
}

func attrValue(se xml.StartElement) string {
	for _, a := range se.Attr {
		if a.Name.Local == "Value" {
			return a.Value
		}
	}
	return ""
}

func attrInt(se xml.StartElement, name string, def int) int {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			if n, err := strconv.Atoi(a.Value); err == nil {
				return n
			}
		}
	}
	return def
}

func atof(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
func atoi(s string) int     { v, _ := strconv.Atoi(s); return v }

func parseSet(r io.Reader) (*Set, error) {
	dec := xml.NewDecoder(r)
	s := &Set{}
	var stack []string

	var (
		track     *Track
		clip      *Clip
		clipDepth int
		loc       *Locator
	)
	inLiveSet := func() bool { return len(stack) >= 2 && stack[1] == "LiveSet" }

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			stack = append(stack, name)
			n := len(stack)
			if !inLiveSet() {
				continue
			}
			// Tracks: Ableton/LiveSet/Tracks/<Track>
			if n == 4 && stack[2] == "Tracks" {
				switch name {
				case "MidiTrack", "AudioTrack", "GroupTrack":
					kind := map[string]string{"MidiTrack": "midi", "AudioTrack": "audio", "GroupTrack": "group"}[name]
					track = &Track{Kind: kind, GroupID: -1, ID: attrInt(t, "Id", 0)}
				}
			}
			if track != nil && n > 4 && stack[2] == "Tracks" {
				switch {
				case n == 6 && stack[4] == "Name" && name == "EffectiveName":
					track.Name = attrValue(t)
				case n == 5 && name == "Color":
					track.Color = liveColor(atoi(attrValue(t)))
				case n == 5 && name == "TrackGroupId":
					track.GroupID = atoi(attrValue(t))
				}
				if clip == nil && (name == "MidiClip" || name == "AudioClip") && n >= 3 &&
					stack[n-2] == "Events" && stack[n-3] == "ArrangerAutomation" {
					clip = &Clip{Color: track.Color}
					clipDepth = n
				} else if clip != nil && n == clipDepth+1 {
					switch name {
					case "CurrentStart":
						clip.Start = atof(attrValue(t))
					case "CurrentEnd":
						clip.End = atof(attrValue(t))
					case "Name":
						clip.Name = attrValue(t)
					case "Color":
						clip.Color = liveColor(atoi(attrValue(t)))
					}
				}
			}
			// Locators: Ableton/LiveSet/Locators/Locators/Locator
			if n == 5 && name == "Locator" && stack[2] == "Locators" {
				loc = &Locator{}
			} else if loc != nil && n == 6 {
				switch name {
				case "Time":
					loc.Time = atof(attrValue(t))
				case "Name":
					loc.Name = attrValue(t)
				}
			}
			// Transport: Ableton/LiveSet/Transport/<x>
			if n == 4 && stack[2] == "Transport" {
				switch name {
				case "LoopOn":
					s.Loop.On = attrValue(t) == "true"
				case "LoopStart":
					s.Loop.Start = atof(attrValue(t))
				case "LoopLength":
					s.Loop.Length = atof(attrValue(t))
				case "CurrentTime":
					s.Playhead = atof(attrValue(t))
				}
			}
		case xml.EndElement:
			n := len(stack)
			name := t.Name.Local
			if clip != nil && n == clipDepth {
				if clip.End > clip.Start {
					track.Clips = append(track.Clips, *clip)
					if clip.End > s.Length {
						s.Length = clip.End
					}
				}
				clip = nil
			}
			if track != nil && n == 4 && stack[2] == "Tracks" {
				sort.Slice(track.Clips, func(i, j int) bool { return track.Clips[i].Start < track.Clips[j].Start })
				s.Tracks = append(s.Tracks, *track)
				track = nil
			}
			if loc != nil && n == 5 && name == "Locator" {
				s.Locators = append(s.Locators, *loc)
				loc = nil
			}
			stack = stack[:n-1]
		}
	}
	return s, nil
}

// FindNewestSet returns the most recently modified .als under root
// (Sets/<project>/<name>.als), skipping Backup folders.
func FindNewestSet(root string) (string, error) {
	var best string
	var bestT int64
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == "Backup" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".als") {
			if info, e := d.Info(); e == nil && info.ModTime().UnixNano() > bestT {
				best, bestT = p, info.ModTime().UnixNano()
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if best == "" {
		return "", os.ErrNotExist
	}
	return best, nil
}
