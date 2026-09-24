package main

import "testing"

func TestChordFiresOnceWhenBothHeld(t *testing.T) {
	c := newChordDetector()
	if c.onCC(ccSession, 127) {
		t.Fatal("session alone must not fire")
	}
	if !c.onCC(ccShift, 127) {
		t.Fatal("shift+session must fire")
	}
	if c.onCC(ccShift, 127) {
		t.Fatal("debounce must block repeat")
	}
}

func TestChordReleaseClears(t *testing.T) {
	c := newChordDetector()
	c.onCC(ccShift, 127)
	c.onCC(ccShift, 0)
	if c.onCC(ccSession, 127) {
		t.Fatal("released shift must not count")
	}
}
