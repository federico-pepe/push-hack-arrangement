package main

import (
	"os"
	"strings"
	"testing"
)

const sampleXML = `<Ableton><LiveSet>
<Tracks>
 <MidiTrack Id="5"><Name><EffectiveName Value="Kick"/></Name><Color Value="9"/><TrackGroupId Value="-1"/>
  <DeviceChain><MainSequencer>
   <ClipSlotList><ClipSlot><Value><MidiClip Id="1"><CurrentStart Value="0"/><CurrentEnd Value="99"/></MidiClip></Value></ClipSlot></ClipSlotList>
   <ClipTimeable><ArrangerAutomation><Events>
     <MidiClip Id="2" Time="8"><CurrentStart Value="8"/><CurrentEnd Value="16"/><Name Value="A"/><Color Value="0"/><Loop><LoopStart Value="0"/></Loop></MidiClip>
     <MidiClip Id="3" Time="0"><CurrentStart Value="0"/><CurrentEnd Value="4"/><Name Value="B"/></MidiClip>
   </Events></ArrangerAutomation></ClipTimeable>
  </MainSequencer></DeviceChain></MidiTrack>
 <ReturnTrack Id="6"><Name><EffectiveName Value="Rev"/></Name></ReturnTrack>
</Tracks>
<Locators><Locators><Locator Id="0"><Time Value="4"/><Name Value="Drop"/></Locator></Locators></Locators>
<Transport><LoopOn Value="true"/><LoopStart Value="8"/><LoopLength Value="16"/><CurrentTime Value="2.5"/></Transport>
</LiveSet></Ableton>`

func TestParseSample(t *testing.T) {
	s, err := parseSet(strings.NewReader(sampleXML))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tracks) != 1 {
		t.Fatalf("tracks=%d want 1 (return skipped)", len(s.Tracks))
	}
	tr := s.Tracks[0]
	if tr.Name != "Kick" || tr.Kind != "midi" || len(tr.Clips) != 2 {
		t.Fatalf("bad track %+v", tr)
	}
	if tr.Clips[0].Start != 0 || tr.Clips[1].Name != "A" || tr.Clips[1].Color != liveColors[0] {
		t.Fatalf("bad clips %+v", tr.Clips)
	}
	if tr.Clips[0].Color != liveColors[9] {
		t.Fatalf("clip must inherit track colour: %+v", tr.Clips[0])
	}
	if s.Length != 16 || len(s.Locators) != 1 || s.Locators[0].Name != "Drop" {
		t.Fatalf("bad set %+v", s)
	}
	if !s.Loop.On || s.Loop.Start != 8 || s.Loop.Length != 16 || s.Playhead != 2.5 {
		t.Fatalf("bad transport %+v", s)
	}
}

// Real set is not committed (private project). Skipped when absent.
func TestParseRealSet(t *testing.T) {
	const p = "../testdata/p3.als"
	if _, err := os.Stat(p); err != nil {
		t.Skip("no testdata/p3.als")
	}
	s, err := LoadSetFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tracks) < 10 || s.Length <= 0 {
		t.Fatalf("tracks=%d length=%v", len(s.Tracks), s.Length)
	}
	t.Logf("%s: %d tracks, %.0f beats, %d locators", s.Name, len(s.Tracks), s.Length, len(s.Locators))
}
