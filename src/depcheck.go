package main

// Display hack: needs push-manager + push-display. Log state changes only.

import (
	"log"
	"net/http"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/pmclient"
)

type depState int

const (
	depUnknown depState = iota
	depPushManagerUnreachable
	depPushDisplayNotAttached
	depOK
)

func runDependencyWatcher(pmURL string) {
	client := &pmclient.Client{Base: pmURL, HTTP: &http.Client{Timeout: 2 * time.Second}}
	last := depUnknown
	check := func() {
		state := checkDependency(client)
		if state == last {
			return
		}
		last = state
		switch state {
		case depPushManagerUnreachable:
			log.Printf("WARNING: push-manager not reachable at %s - Arrangement Mode cannot draw until it runs.", pmURL)
		case depPushDisplayNotAttached:
			log.Printf("WARNING: push-manager up but push-display framebuffer not connected - install push-display.")
		case depOK:
			log.Printf("push-manager + push-display OK")
		}
	}
	check()
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		check()
	}
}

func checkDependency(c *pmclient.Client) depState {
	st, err := c.DisplayStatus()
	if err != nil {
		return depPushManagerUnreachable
	}
	if !st.Connected {
		return depPushDisplayNotAttached
	}
	return depOK
}
