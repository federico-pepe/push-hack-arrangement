package main

import (
	"os"
	"testing"
	"time"
)

func TestNextBackoff(t *testing.T) {
	b := restartMin
	for i := 0; i < 10; i++ {
		b = nextBackoff(b, time.Second) // keeps crashing fast
	}
	if b != restartMax {
		t.Fatalf("must cap at %v, got %v", restartMax, b)
	}
	if got := nextBackoff(b, 2*healthyAfter); got != restartMin {
		t.Fatalf("healthy run must reset, got %v", got)
	}
	if got := nextBackoff(restartMin, time.Second); got != 2*time.Second {
		t.Fatalf("doubles, got %v", got)
	}
}

func TestModeStateFile(t *testing.T) {
	markModeOn()
	releaseIfLeftOn("http://127.0.0.1:1") // unreachable: must not hang or panic, must clear the file
	if _, err := osStat(modeStateFile); err == nil {
		t.Fatal("state file must be removed after release")
	}
}

func osStat(p string) (interface{}, error) { return os.Stat(p) }
