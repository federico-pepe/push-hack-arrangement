package main

// Arrangement Mode state. ON = display takeover + MIDI intercept.
// OFF or exit must always release both, or Push is left dead.

import (
	"image"
	"log"
	"sync"

	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
)

type modeCtl struct {
	pm    *pmclient.Client
	frame func() image.Image // current view, built on demand

	mu sync.Mutex
	on bool
}

func newModeCtl(pm *pmclient.Client, frame func() image.Image) *modeCtl {
	return &modeCtl{pm: pm, frame: frame}
}

func (m *modeCtl) push() {
	if err := m.pm.PushImage(m.frame()); err != nil {
		log.Printf("display: push frame: %v", err)
	}
}

// toggle flips the mode. OFF is instant: no message, no wait.
func (m *modeCtl) toggle() {
	m.mu.Lock()
	m.on = !m.on
	on := m.on
	m.mu.Unlock()

	if !on {
		m.release()
		log.Print("Arrangement Mode OFF")
		return
	}
	if err := m.pm.SetMode(2); err != nil {
		log.Printf("display: enable takeover: %v", err)
	}
	if err := m.pm.SetMidiFilter(true); err != nil {
		log.Printf("display: enable midi filter: %v", err)
	}
	m.push()
	log.Print("Arrangement Mode ON (takeover + MIDI intercept)")
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

// refresh redraws if the mode is on (data changed).
func (m *modeCtl) refresh() {
	m.mu.Lock()
	on := m.on
	m.mu.Unlock()
	if on {
		m.push()
	}
}
