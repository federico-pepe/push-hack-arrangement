package main

// Arrangement Mode state. ON = display takeover + MIDI intercept.
// OFF or exit must always release both, or Push is left dead.

import (
	"image"
	"log"
	"sync"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
)

const osdDuration = 1500 * time.Millisecond

type modeCtl struct {
	pm *pmclient.Client

	mu    sync.Mutex
	on    bool
	epoch int // bumped per toggle; stale timers check it
}

func newModeCtl(pm *pmclient.Client) *modeCtl { return &modeCtl{pm: pm} }

func (m *modeCtl) push(img image.Image) {
	if err := m.pm.PushImage(img); err != nil {
		log.Printf("display: push frame: %v", err)
	}
}

// toggle flips the mode. Runs in its own goroutine (HTTP calls).
func (m *modeCtl) toggle() {
	m.mu.Lock()
	m.on = !m.on
	on := m.on
	m.epoch++
	epoch := m.epoch
	m.mu.Unlock()

	if on {
		if err := m.pm.SetMode(2); err != nil {
			log.Printf("display: enable takeover: %v", err)
		}
		if err := m.pm.SetMidiFilter(true); err != nil {
			log.Printf("display: enable midi filter: %v", err)
		}
		m.push(renderWithOSD(renderTestFrame(), "Arrangement Mode: ON"))
		log.Print("Arrangement Mode ON (takeover + MIDI intercept)")
		// After the OSD time, show the plain frame - unless toggled again.
		time.AfterFunc(osdDuration, func() {
			m.mu.Lock()
			still := m.on && m.epoch == epoch
			m.mu.Unlock()
			if still {
				m.push(renderTestFrame())
			}
		})
		return
	}

	// OFF: show message while still in takeover, then give everything back.
	m.push(renderWithOSD(renderTestFrame(), "Arrangement Mode: OFF"))
	log.Print("Arrangement Mode OFF")
	time.AfterFunc(osdDuration, func() {
		m.mu.Lock()
		still := !m.on && m.epoch == epoch
		m.mu.Unlock()
		if still {
			m.release()
		}
	})
}

// release hands display + MIDI back to the native Push UI / Live.
func (m *modeCtl) release() {
	if err := m.pm.SetMidiFilter(false); err != nil {
		log.Printf("display: disable midi filter: %v", err)
	}
	if err := m.pm.SetMode(0); err != nil {
		log.Printf("display: disable takeover: %v", err)
	}
}
