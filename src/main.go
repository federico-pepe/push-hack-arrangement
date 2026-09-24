// arrangement: draws the Live Arrangement on the Push 3 screen.
// Shift+Session toggles Arrangement Mode (display takeover + MIDI intercept).
package main

import (
	"flag"
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
	preview := flag.String("preview", "", "write the test frame to this PNG and exit (no device needed)")
	flag.Parse()

	if *preview != "" {
		f, err := os.Create(*preview)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, renderWithOSD(renderTestFrame(), "Arrangement Mode: ON")); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s", *preview)
		return
	}

	// Cold-boot USB-A window: no /dev/snd access before uptime >= 30 s.
	alsaseq.WaitForBootSettle()

	client := pmclient.New(*pm)
	mode := newModeCtl(client)
	go runDependencyWatcher(*pm)

	stop := make(chan struct{})
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
