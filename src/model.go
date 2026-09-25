package main

// Arrangement data, independent of where it came from (.als file now,
// Remote Script socket later).

type Clip struct {
	Start, End float64 // beats
	Name       string
	Color      uint32 // 0xRRGGBB
}

type Track struct {
	Name    string
	Kind    string // "midi", "audio", "group"
	Color   uint32
	GroupID int // id of the parent group track, -1 if none
	ID      int
	Clips   []Clip
}

type Locator struct {
	Time float64
	Name string
}

type Loop struct {
	On            bool
	Start, Length float64
}

type Set struct {
	Name        string
	Tracks      []Track
	Locators    []Locator
	Loop        Loop
	Playhead    float64 // saved insert marker (beats)
	Length      float64 // end of last clip (beats)
	BeatsPerBar float64 // 0 = 4
	Overridden  bool    // Session clips override the arrangement (Back to Arrangement lit): draw dimmed
}
