package main

// Shift (CC49) + Session (CC51) chord. Same shape as braids/keyboard-visualizer:
// held set of CCs, 500 ms debounce.

import (
	"sync"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/push3"
)

const (
	ccShift   = uint8(push3.CCShift)
	ccSession = uint8(push3.CCSession)

	chordDebounce = 500 * time.Millisecond
)

type chordDetector struct {
	mu       sync.Mutex
	held     map[uint8]bool
	lastFire time.Time
}

func newChordDetector() *chordDetector {
	return &chordDetector{held: map[uint8]bool{}}
}

// onCC tracks held CCs. Returns true once when Shift+Session go down together.
func (c *chordDetector) onCC(cc, val byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if val > 0 {
		c.held[cc] = true
	} else {
		delete(c.held, cc)
	}
	if !(c.held[ccShift] && c.held[ccSession]) {
		return false
	}
	now := time.Now()
	if now.Sub(c.lastFire) < chordDebounce {
		return false
	}
	c.lastFire = now
	return true
}

// isHeld reports whether a CC is currently down.
func (c *chordDetector) isHeld(cc uint8) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.held[cc]
}
