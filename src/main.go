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
)

func main() {
	pm := flag.String("pm", "http://127.0.0.1:7701", "push-manager base URL")
	setPath := flag.String("set", "", "Live Set (.als) to show (default: newest under "+setsRoot+")")
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
		if err := png.Encode(f, renderArrangement(s, fitView(s))); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s (%d tracks)", *preview, len(s.Tracks))
		return
	}

	// Cold-boot USB-A window: no /dev/snd access before uptime >= 30 s.
	alsaseq.WaitForBootSettle()

	client := pmclient.New(*pm)
	go runDependencyWatcher(*pm)

	st := &store{}
	frame := func() image.Image {
		s, msg := st.get()
		if s == nil {
			return renderMessage(msg)
		}
		return renderArrangement(s, fitView(s))
	}
	mode := newModeCtl(client, frame)

	stop := make(chan struct{})
	go watchSets(st, *setPath, mode.refresh, stop)
	h := &midiHandler{chord: newChordDetector(), onFire: mode.toggle}
	go runMIDI(h, stop)

	log.Print("arrangement ready: press Shift+Session on Push")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Print("stopping: releasing display + MIDI intercept")
	close(stop)
	mode.release() // always, whatever the last state was
}
