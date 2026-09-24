// arrangement: draws the Live Arrangement on the Push 3 screen.
// Milestone 1: static test frame in display takeover.
package main

import (
	"flag"
	"image/png"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
)

func main() {
	pm := flag.String("pm", "http://127.0.0.1:7701", "push-manager base URL")
	preview := flag.String("preview", "", "write the test frame to this PNG and exit (no device needed)")
	flag.Parse()

	frame := renderTestFrame()

	if *preview != "" {
		f, err := os.Create(*preview)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, frame); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s", *preview)
		return
	}

	c := pmclient.New(*pm)
	if st, err := c.DisplayStatus(); err != nil {
		log.Fatalf("push-manager unreachable: %v", err)
	} else if !st.Connected {
		log.Fatal("push-manager up but push-display framebuffer not connected")
	}
	if err := c.SetMode(2); err != nil {
		log.Fatalf("takeover: %v", err)
	}
	// Always give the screen back, even on signal.
	defer func() {
		if err := c.SetMode(0); err != nil {
			log.Printf("release display: %v", err)
		}
	}()
	if err := c.PushImage(frame); err != nil {
		log.Printf("push frame: %v", err)
	}
	log.Print("test frame shown; Ctrl+C to release")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
