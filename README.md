# Push Hack: Arrangement

Shows the Live Arrangement on the Push 3 screen. Zoom in time and in track height.
Press **Shift + Session** to turn Arrangement Mode on or off. The second press
closes the view at once, with no message and no wait.

While the mode is on, the hack uses push-manager's MIDI intercept. Live does not
see the Push buttons, pads and encoders. Every LED the hack does not use goes dark.

**Status:** milestone 5, first part done. A Remote Script inside Live sends the
arrangement of the **open** set (also unsaved edits), the playhead, and the loop
and punch state to the hack. See
[plans/2026-09-24-arrangement-hack.md](plans/2026-09-24-arrangement-hack.md).

The view shows one lane per track, clips in their colours, locators (sky-blue
lines; the one you last jumped to is white), the loop (a yellow bar on the
ruler, with full-height yellow lines at its start and end while Loop is on),
the play-start anchor (a small green triangle) and the playhead (white line).
Clips draw dimmed while Session clips override the arrangement, and return to
full colour once you press Back to Arrangement.

## Controls (while Arrangement Mode is on)

| Control | Action |
|---|---|
| Shift + Session | Turn the mode on or off |
| Jog wheel | Move a scrub marker to look around, without touching Live's actual playback (no audible jump) |
| Shift + jog wheel | Really seek: moves Live's playhead for real, same as before (audible if playing) |
| Jog wheel press | Set the play-start anchor at the current position, silently (nothing jumps). Pressing again at that SAME spot clears it instead -- no Shift needed |
| Play | Start or stop Live. Starting jumps to the anchor first, if one is set. The button is white when stopped, green while playing |
| Session | Back to Arrangement. Lights bright white when Session clips override the arrangement (the button has a greyscale LED) |
| Page < / Page > | Jump to the previous / next locator (a real seek). The one you land on turns white on screen. Both buttons are always lit white |
| Top button 1 | Loop on/off. White when off, yellow when on |
| Top button 2 | Punch In on/off. White when off, yellow when on |
| Top button 3 | Punch Out on/off. White when off, yellow when on |
| Encoder 1 (touch-capable) | Loop start, one bar per detent (Shift = a precise 1/16 beat step). Has a bit of resistance built in (a few raw ticks per step), so a light touch does not send it flying. Touching it shows the exact position (bars.beats.16th) over the arrangement |
| Encoder 2 (touch-capable) | Loop length, one bar per detent (Shift = a precise 1/16 beat step). Same resistance as Encoder 1. Touching it shows the exact length the same way |
| Volume knob | Scroll the view in time |
| Tempo knob | Zoom. It zooms in time at first, always keeping the playhead (or the scrub marker, if one is active) centred on screen. Press the knob to switch between time zoom and track zoom. A short message shows the mode. When lanes are tall enough, track and clip names appear |
| Delete + Tempo press | Reset the zoom: shows the whole song, every track, from scratch |
| D-pad up / down | Scroll tracks. Hold to repeat. Lit white only while that direction has more to scroll |
| D-pad left / right | Scroll a quarter screen in time. Hold to repeat. Lit white only while that direction has more to scroll |

Touching the jog wheel or encoders 1/2 shows a bar at the bottom of the screen
with the exact playhead position or loop start/length (bars.beats.16th, e.g.
`97.1.1`, or `16.0.0` for a 16-bar length) — see
[docs/touch-layer.md](docs/touch-layer.md).

While the song plays, the view scrolls so the playhead stays near the left. It
stops following for 2 seconds after you move the view by hand.

## Requirements

- push-manager and push-display installed and running on the Push.
- The PushHack Remote Script. It comes with Push Hack (`install.sh` or the Push Hack update puts it in
  Live's User Library). Enable it once: Live Preferences, a free control-surface slot, select
  `PushHack`, Input and Output = None. Then restart Live. Live's `Log.txt` should show
  "PushHack alive". Push Manager's System page shows if it runs.
- If the old `PushHackArrangement` script is still in the User Library, remove it (System page, Remove).

## Try it

```bash
scripts/deploy.sh            # builds, copies to push.local, runs in the foreground
```

The hack has no boot service. It runs until the Push restarts; after a restart, run `scripts/deploy.sh` again (or stop the process, `scp` the binary to `/data/push-hack/hacks/arrangement/`, and start it in the background as root). Then press Shift + Session on the Push. Press it again to switch off. Ctrl+C stops the
hack and gives the screen and MIDI back.

The hack talks to the Remote Script over `/tmp/push-hack.sock`. Tests: `make test`.

The same Remote Script also serves `pta-module-arrangement`, the version for a tethered Push (private repo). See `docs/protocol.md`.

Without a device, `make preview` writes `build/preview.png` from `testdata/p3.als` (not in git; use your own set).

## Performance and safety

- With a 40-track, 342-clip set, drawing at about 8 frames per second used about 6% of one core in the hack and 4% in push-manager. Memory: 6 MB for the supervisor and about 12 MB for the working process.
- If the hack is killed while the mode is on, the supervisor gives the display and MIDI back and restarts the hack.

## Roadmap

Not done yet:

- Touch layer for the other knobs (Volume, Tempo, the other top knobs): built so far for the jog wheel and loop knobs 1/2. Design: [docs/touch-layer.md](docs/touch-layer.md). Add its content whenever a new control is added.
- Record. Metronome.
- Jog step buttons (finer and coarser) from the reference script.
- Editing: nudge, resize, split, duplicate, spacer. These are from the reference script and are outside v1.
- Send only the visible window for a very huge set (the Remote Script already slows down snapshots on big sets).
