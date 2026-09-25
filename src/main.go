// arrangement: draws the Live Arrangement on the Push 3 screen.
// Shift+Session toggles Arrangement Mode (display takeover + MIDI intercept).
package main

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/federico-pepe/ableton-push-hack/core/alsaseq"
	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

func main() {
	pm := flag.String("pm", "http://127.0.0.1:7701", "push-manager base URL")
	// The catalog's boot service starts every hack as `<binary> -config <hack.json>`.
	// This hack has no settings of its own, so the flag is accepted and ignored.
	_ = flag.String("config", "", "path to hack.json (passed by the boot service; unused)")
	setPath := flag.String("set", "", "dev: show this saved Live Set (.als) instead of the Remote Script data")
	hz := flag.Int("hzoom", 0, "preview: horizontal zoom ticks")
	vz := flag.Int("vzoom", 0, "preview: vertical zoom ticks")
	scroll := flag.Int("scroll", 0, "preview: scroll ticks in time")
	tscroll := flag.Int("tracks", 0, "preview: scroll tracks")
	bench := flag.Int("bench", 0, "dev: redraw for this many seconds with a fake playhead and print CPU use (use with -set)")
	preview := flag.String("preview", "", "write the view to this PNG and exit (needs -set, no device needed)")
	flag.Parse()

	if *preview != "" {
		s, err := LoadSetFile(*setPath)
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(*preview)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, renderArrangement(s, previewView(s, *hz, *vz, *scroll, *tscroll))); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s (%d tracks)", *preview, len(s.Tracks))
		return
	}

	if *bench > 0 {
		s, err := LoadSetFile(*setPath)
		if err != nil {
			log.Fatal(err)
		}
		vc := newViewCtl(func() *Set { return s })
		if err := runBench(pmclient.New(*pm), s, vc, *bench); err != nil {
			log.Fatal(err)
		}
		return
	}

	// The boot service starts this parent; the real work runs as a supervised child.
	if os.Getenv(supervisedEnv) != "1" {
		runSupervisor(*pm)
		return
	}

	// Cold-boot USB-A window: no /dev/snd access before uptime >= 30 s.
	alsaseq.WaitForBootSettle()

	client := pmclient.New(*pm)
	go runDependencyWatcher(*pm)

	st := &store{}
	vc := newViewCtl(func() *Set { s, _, _ := st.get(); return s })
	frame := func() image.Image {
		s, msg, hint := st.get()
		if s == nil {
			return renderMessage(msg, hint)
		}
		return renderArrangement(s, vc.viewport(s))
	}
	mode := newModeCtl(client, frame)

	stop := make(chan struct{})
	if *setPath != "" {
		// dev: show a saved .als once instead of the Remote Script
		s, err := LoadSetFile(*setPath)
		if err != nil {
			log.Fatal(err)
		}
		st.putSet(s)
	} else {
		var lastPlaying, lastBTA bool
		go runLiveSource(st, func() {
			if mode.isOn() {
				t, playing := st.pos()
				vc.follow(t, playing)
				playing, bta := st.state()
				if playing != lastPlaying || bta != lastBTA {
					lastPlaying, lastBTA = playing, bta
					syncStateLEDs(playing, bta)
					burstLEDs(func() { syncStateLEDs(st.state()) })
				}
			}
			mode.refresh()
		}, stop)
	}
	chord := newChordDetector()
	ctl := newControls(vc, st, mode.isOn, func() bool { return chord.isHeld(ccShift) }, mode.refresh, sendCmd,
		func(what string) {
			if what == "play" {
				// Show the new state at once, before Live confirms; Live's own LED
				// update would otherwise flash another colour first.
				playing, _ := st.state()
				setLED(byte(push3.CCPlay), map[bool]byte{true: ledPlayWhite, false: ledPlayGreen}[playing])
			}
			burstLEDs(func() { syncStateLEDs(st.state()) })
		})
	h := &midiHandler{chord: chord, onCC: ctl.onCC,
		onFire: func() {
			mode.toggle()
			if mode.isOn() {
				syncStateLEDs(st.state())
			} else {
				ctl.releaseAll()
			}
		}}
	go runMIDI(h, stop)

	log.Print("arrangement ready: press Shift+Session on Push")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Print("stopping: releasing display + MIDI intercept")
	close(stop)
	mode.release() // always, whatever the last state was
}

// previewView applies zoom/scroll ticks to a fit view, for `make preview` screenshots.
func previewView(s *Set, hz, vz, scroll, tracks int) viewport {
	vc := newViewCtl(func() *Set { return s })
	vc.viewport(s)
	vc.zoomH(hz)
	vc.zoomV(vz)
	vc.scrollT(scroll)
	vc.scrollTracks(tracks)
	return vc.viewport(s)
}
