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

type modeCtl struct {
	pm    *pmclient.Client
	frame func() image.Image // current view, built on demand

	mu   sync.Mutex
	on   bool
	quit chan struct{} // closed on OFF; stops the LED blackout loop

	lastPush time.Time
	pending  bool
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
		ledActive.Store(false)
		m.mu.Lock()
		if m.quit != nil {
			close(m.quit)
			m.quit = nil
		}
		m.mu.Unlock()
		m.release()
		log.Print("Arrangement Mode OFF")
		return
	}
	resetLit()
	ledActive.Store(true)
	blackoutLEDs()
	m.mu.Lock()
	m.quit = make(chan struct{})
	quit := m.quit
	m.mu.Unlock()
	go keepDark(quit)
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

// minPushGap caps redraws (each one is a PNG encode + HTTP post).
const minPushGap = 120 * time.Millisecond

// refresh redraws if the mode is on. Rate limited; a trailing redraw is
// scheduled so the last change is never lost.
func (m *modeCtl) refresh() {
	m.mu.Lock()
	if !m.on {
		m.mu.Unlock()
		return
	}
	wait := minPushGap - time.Since(m.lastPush)
	if wait > 0 {
		if !m.pending {
			m.pending = true
			time.AfterFunc(wait, func() {
				m.mu.Lock()
				m.pending = false
				m.mu.Unlock()
				m.refresh()
			})
		}
		m.mu.Unlock()
		return
	}
	m.lastPush = time.Now()
	m.mu.Unlock()
	m.push()
}

// keepDark repeats the blackout: Live may still repaint a few LEDs.
func keepDark(quit <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-quit:
			return
		case <-t.C:
			blackoutLEDs()
		}
	}
}

func (m *modeCtl) isOn() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.on
}
