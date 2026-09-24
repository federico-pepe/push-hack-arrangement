package main

// MIDI input: own ALSA seq port subscribed to Push 3's Live port (16:0).
// Sees every event even while the intercept filter hides them from Live.

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/federico-pepe/ableton-push-hack/core/alsaseq"
)

const midiPortName = "Arrangement MIDI In"

type midiHandler struct {
	chord  *chordDetector
	onFire func()
}

func (h *midiHandler) Fixed(evType uint8, src alsaseq.Addr, data []byte) {
	if evType != alsaseq.EvController {
		return
	}
	if src.Client != alsaseq.Push3ClientDefault || src.Port != alsaseq.Push3PortDefault {
		return
	}
	// Control surface CCs are channel 0. Other channels = per-pad MPE stream.
	if data[0]&0x0F != 0 {
		return
	}
	cc := uint8(binary.LittleEndian.Uint32(data[4:]) & 0x7F)
	val := uint8(binary.LittleEndian.Uint32(data[8:]) & 0x7F)
	if h.chord.onCC(cc, val) {
		go h.onFire()
	}
}

func (h *midiHandler) VarLen(uint8, alsaseq.Addr, []byte) {}

// runMIDI opens the port and subscribes; retries until it works or stop closes.
func runMIDI(h alsaseq.Handler, stop <-chan struct{}) {
	pinned := alsaseq.Addr{Client: alsaseq.Push3ClientDefault, Port: alsaseq.Push3PortDefault}
	for {
		seq, err := alsaseq.Open()
		if err == nil {
			_, err = seq.CreatePort(midiPortName,
				alsaseq.CapWrite|alsaseq.CapSubsWrite, alsaseq.PortTypeMidi|alsaseq.PortTypeApp)
			if err == nil {
				err = seq.Subscribe(pinned)
			}
			if err == nil {
				log.Printf("subscribed to Push 3 %v", pinned)
				done := make(chan struct{})
				go func() {
					if e := seq.ReadLoop(h); e != nil {
						log.Printf("midi read loop ended: %v", e)
					}
					close(done)
				}()
				select {
				case <-stop:
					seq.Close()
					return
				case <-done:
					seq.Close() // reopen
				}
			} else {
				seq.Close()
			}
		}
		if err != nil {
			log.Printf("midi setup: %v - retry in 3s", err)
		}
		select {
		case <-stop:
			return
		case <-time.After(3 * time.Second):
		}
	}
}
