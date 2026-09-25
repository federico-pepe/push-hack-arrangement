package main

// Supervisor: the process the boot service starts is a small parent. It runs the
// real program as a child (env ARR_SUPERVISED=1) and restarts it if it dies.
// If the child dies while Arrangement Mode is on (kill -9, crash), the parent
// gives the display and MIDI back, so Push is never left dead.
//
// The child marks "mode on" with a state file; the parent reads it after a death.

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
)

const (
	supervisedEnv = "ARR_SUPERVISED"
	modeStateFile = "/tmp/push-hack-arrangement.mode"

	restartMin   = time.Second
	restartMax   = 30 * time.Second
	healthyAfter = 30 * time.Second // ran this long: next restart is fast again
	stopWait     = 5 * time.Second  // after SIGTERM, then SIGKILL
)

func markModeOn()  { _ = os.WriteFile(modeStateFile, []byte("1"), 0o600) }
func markModeOff() { _ = os.Remove(modeStateFile) }

// releaseIfLeftOn hands display and MIDI back if the child left the mode on.
func releaseIfLeftOn(pmURL string) {
	if _, err := os.Stat(modeStateFile); err != nil {
		return
	}
	log.Print("supervisor: child ended with Arrangement Mode on - releasing display and MIDI intercept")
	c := pmclient.New(pmURL)
	if err := c.SetMidiFilter(false); err != nil {
		log.Printf("supervisor: disable midi filter: %v", err)
	}
	if err := c.SetMode(0); err != nil {
		log.Printf("supervisor: disable takeover: %v", err)
	}
	markModeOff()
}

func nextBackoff(cur, ran time.Duration) time.Duration {
	if ran >= healthyAfter {
		return restartMin
	}
	cur *= 2
	if cur > restartMax {
		cur = restartMax
	}
	return cur
}

func runSupervisor(pmURL string) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	backoff := restartMin
	for {
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), supervisedEnv+"=1")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		start := time.Now()
		if err := cmd.Start(); err != nil {
			log.Printf("supervisor: cannot start child: %v", err)
		} else {
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case s := <-sig:
				// The service only signals this PID: pass it on and wait for the child.
				_ = cmd.Process.Signal(s)
				select {
				case <-done:
				case <-time.After(stopWait):
					_ = cmd.Process.Kill()
					<-done
				}
				releaseIfLeftOn(pmURL)
				return
			case err := <-done:
				log.Printf("supervisor: child exited after %v: %v", time.Since(start).Round(time.Second), err)
			}
			releaseIfLeftOn(pmURL)
		}
		backoff = nextBackoff(backoff, time.Since(start))
		log.Printf("supervisor: restarting in %v", backoff)
		select {
		case <-sig:
			return
		case <-time.After(backoff):
		}
	}
}
