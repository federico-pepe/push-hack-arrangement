# Changelog

## [Unreleased]

## [0.0.3-alpha] - 2026-09-25

- Volume knob now scrolls the view in time (this replaces Shift + jog wheel, which is gone; the jog wheel always moves the playhead).
- Tempo knob zooms. Pressing it (CC15) switches between time zoom and track zoom. A short message ("ZOOM: TIME" or "ZOOM: TRACKS") shows for about a second.
- Docs: `docs/touch-layer.md`, the design for a temporary bar shown when you touch a knob or the jog wheel (not built yet).

## [0.0.2-alpha] - 2026-09-25

- Project created. Plan added.
- Milestone 1: Go skeleton draws a static 960x160 test frame in display takeover; `-preview` writes a PNG without a device.
- Milestone 2: Shift+Session (CC49+CC51) toggles Arrangement Mode: display takeover + MIDI intercept, ON/OFF message on screen, release on exit. `scripts/deploy.sh` runs it on the Push.
- `core` now taken from GitHub (core v0.2.0), no local replace.
- Toggle changed: Shift+Session shows the view (title "Arrangement Mode"); the second press closes it at once, no message.
- Milestone 3: PushHackArrangement Remote Script streams the open set (tracks, clips, exact colours, locators, loop, time signature, playhead) over a Unix socket. The hack draws it fit to the screen, with a moving playhead. The saved-file (.als) reader is now only for `make preview` and `-set`.
- All button LEDs and the pad grid go dark while Arrangement Mode is on.
- Removed the "Arrangement Mode" title from the view.
- Milestone 4: horizontal zoom (Volume dial), vertical zoom (Tempo dial), scroll in time (jog wheel, D-pad left/right), scroll tracks (D-pad up/down). Track and clip names show when lanes are tall enough. The view follows the playhead while playing.
- All LEDs go dark in the mode (every CC, so the scene buttons too). Play is green while playing.
- Jog wheel now moves the playhead (grid by zoom, speed ramp). Shift + jog scrolls the view.
- Play starts and stops Live. Session = Back to Arrangement (bright white, palette 120, when available; the button has a greyscale LED). Play is palette 126. New Remote Script commands: `set_time`, `play_toggle`, `bta`. **Restart Live after updating the script.**
- D-pad repeats while held. The view scrolls smoothly with the playhead during playback.
- Accept `-config <hack.json>`: the catalog's boot service always passes it, and without it the hack would exit at boot. LEDs: Play = palette 126, Back to Arrangement = palette 4.
- Clips are drawn dimmed while Session clips override the arrangement (Back to Arrangement available); crisp again after Back to Arrangement. Play is white (120) when stopped.
- Less LED flicker: pressing Play shows the new colour at once (white to green, green to white). After Play, Back to Arrangement, or a change of state reported by Live, the LED blackout repeats for about a second so Live's own repaint is covered quickly.
- Docs: `docs/architecture.md`, `docs/protocol.md`, roadmap in the README.
- Supervisor: the boot service starts a small parent process that runs the hack as a child and restarts it if it dies. If the child dies while Arrangement Mode is on (crash, kill -9), the parent releases the display and MIDI intercept, so Push is never left dead. Tested on the device.
- Remote Script: on big sets it waits longer between snapshots, so building them never uses more than about 5% of Live's time.
- `-bench <seconds>` (with `-set`): measure CPU on the device. Result with a 40-track, 342-clip set at about 7.6 frames per second: hack 5.7% of one core, push-manager 4%.
