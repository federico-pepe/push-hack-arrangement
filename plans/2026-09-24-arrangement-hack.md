# Plan: "Arrangement" hack for Push 3

## Context
Push 3 (standalone, full Live) has no arrangement view on its screen. Goal: a new
catalog-style hack that draws the Arrangement on the 960x160 display, with
horizontal + vertical zoom, toggled by Shift+Session with an OSD
"Arrangement Mode: ON/OFF". It must own the hardware via push-manager's
**intercept mode** (push_hook.so neutralises pad/button/encoder MIDI so Live never
sees it) instead of the User-mode Remote Script approach in Push3ArrangementMode.py.
The existing script is the reference for what the Live API can do and for the
control map (CCs), not code to run.

New repo (separate from ableton-push-hack): `~/Developer/push-hack-arrangement`.
Commit identity: **Federico P. only**, never the @ableton account. Set repo-local
`git config user.name "Federico P."` + personal email (copy from
`git config user.email` of the existing push-hack repo, confirm before first
commit; see memory: new repos inherit the work identity).

## Findings that shape the design
- Intercept = shm flag `midiflt` in push_hook.so; toggled via
  `POST /api/midi/filter` (`pmclient.SetMidiFilter`). SysEx always passes.
- Hack subscribes itself to ALSA `16:0` via `core/alsaseq` (button CCs, encoders,
  pads) and calls `SetMidiFilter(true)` while Arrangement Mode is ON.
- Display: `pmclient.SetMode(2)` (takeover) + `PushImage(image.Image)` 960x160.
  Never mmap shm. Needs dependency watcher on `/api/display/status`
  (CLAUDE.md rule). Strings drawn must be ASCII (`core/gfx/text`).
- LEDs: `POST /api/midi/led`; palette in `core/push3`.
- No existing way to read clips/tracks from Live: Browser Bridge (7704) only has
  tempo/play/load. Need new data channel.
- Boot-settle: use `alsaseq.WaitForBootSettle()` (USB-A wedge rule).

## Data source decision
Considered alternatives to a Remote Script: parse the saved Live Set (.als),
Max for Live device, Extensions SDK, MIDI clock/Link. Chosen: **pluggable
`Source` interface in Go** with two implementations:
- `alsSource` (early dev + fallback): reads the newest .als (gzip XML) from
  disk. Static view: tracks, clips, colours, locators, loop. No playhead, stale
  until saved. Lets us build renderer/zoom/intercept with no Live-side code.
- `socketSource` (live path): Remote Script below. Adds playhead, live updates,
  edit commands.
Extensions SDK / M4L not pursued in v1 (unverified on Push standalone).

## Architecture
Two parts in the one repo:

1. **`remote-script/PushHackArrangement/`** (Python, Live MIDI Remote Script,
   read/write bridge, NO controls, no MIDI mapping). Same one-time manual
   activation as Browser Bridge. Uses the Live API seen in the reference script:
   `song.tracks[i].arrangement_clips` (start_time, end_time, name, color),
   `track.color/name`, `song.current_song_time`, `is_playing`, loop
   (`loop_start/loop_length/loop`), `cue_points`, `signature_*`, `tempo`.
   - Snapshot on a slow tick (`update_display` ~10Hz) + listeners for tracks,
     `arrangement_clips` changes; playhead pushed each tick. Send only diffs/
     dirty flags, keep Live CPU low.
   - Transport: **Unix socket** (e.g. `/tmp/push-hack-arrangement.sock`) to avoid
     burning a port in the reserved 7701-7710 block. Line-delimited JSON:
     Live->hack `snapshot`/`playhead`; hack->Live commands
     `set_time`, `play`, `stop`, `set_loop`, `toggle_loop`, `cue_toggle`,
     `cue_jump`, `punch_in/out`.
   - Verify Python version in Live 12 on device and that a Unix socket bind is
     allowed (fallback: 127.0.0.1 dynamic port written to a state file).

2. **`src/` Go binary `arrangement`** (`GOOS=linux GOARCH=amd64`, `hack.json`
   with no web_ui so no port assigned; Makefile like push-manager's). Modules:
   - `midi.go`: alsaseq subscribe 16:0, button/encoder decoding
     (`push3.IsEncoderCC/DecodeRel`), chord detect (Shift CC49 held + Session CC51).
   - `live.go`: client for the socket, holds latest model + reconnect.
   - `model.go`: tracks, clips, loop, cues, playhead.
   - `view.go`: viewport state (time origin, beats-per-px zoom level, lane
     height level, first track), zoom/scroll logic, follow-playhead.
   - `render.go`: draws NRGBA 960x160 with `core/gfx`, `gfx/text`,
     `gfx/widgets`: ruler (bars), track lanes with clip rects in Live colours +
     clipped ASCII names, playhead, loop brace, locators, mode/zoom readout.
     Redraw only when dirty; cap ~15-20 fps; PNG encode cost measured.
   - `osd.go`: 1.5s overlay "Arrangement Mode: ON/OFF" drawn in our frame (ON),
     and for OFF draw it then release takeover after the timer.
   - `mode.go`: state machine. ON: SetMidiFilter(true) + SetMode(2) + start
     render loop. OFF: show OSD, then SetMode(0) + SetMidiFilter(false) and
     restore. Also OFF on shutdown/SIGTERM/crash-safe (defer + watcher so a
     dead hack never leaves the filter on = dead Push).
   - `watch.go`: push-manager/push-display dependency watcher (state-transition
     logging).

## Controls (v1 = view + some editing, from the reference script)
- Shift+Session: toggle mode (ON/OFF + OSD). Chord logic mirrors
  `initMidiChords` in push-manager `midi.go`.
- Volume dial (CC79): horizontal zoom. Tempo dial (CC14): vertical zoom.
- Jog wheel (CC70): scrub AIM (`set_time`), with the script's resolution
  ladder + speed ramp ported (UDB6/UDB7 = finer/coarser). Jog click (CC94)
  = play-start anchor.
- D-pad up/down (CC46/47): select track / scroll tracks; left/right: AIM step.
- Play (CC85) / Record (CC86) / Metronome / Loop (UDB5) / Locator (UDB1) /
  Punch (UDB2/3) / Set Loop (UDB4): ported from the script, sent as commands.
- Nudge/split/duplicate/spacer/etc. from the script: **out of v1** (listed in
  README as roadmap).
- LEDs for the used buttons via `/api/midi/led` while mode is ON; restored on OFF.
- Exact CCs (Session = CC51 confirmed by user, Shift = 49) must be confirmed with MIDI
  monitor on device before coding (`PAD_MIDI_LOGGING`-style).

## Open risks (resolve early on device)
- Session press reaches Live on the first chord press (filter is off until we
  see it): may toggle Live's session view. Mitigate by enabling filter on Shift
  press or accept/restore view; test.
- Filtered events are neutralised but Live's own LEDs/mode remain: confirm
  Live doesn't fight our LEDs.
- PNG-encode CPU at 15-20fps on Push: measure; drop fps/dirty-only render.
- Large sets: cap clips sent (only visible time window + tracks).
- Remote Script activation is manual; document in README.

## Repo scaffolding + docs
`hack.json`, `Makefile`, `src/go.mod` (require+replace to core in the main repo
or a pinned module version), `remote-script/`, `README.md`, `CHANGELOG.md`
(`## [Unreleased]`), `docs/` (protocol, controls, architecture), `plans/`
(this plan copied as `YYYY-MM-DD-arrangement-hack.md`), GitHub Actions release
workflow modelled on push-hack-keyboard-visualizer, `.gitignore`. Later: PR to
`catalog/catalog.json` in ableton-push-hack (separate step, not now).
Writing style: code comments via `caveman` skill; docs via `simple-english`.

## Milestones
1. Repo + identity + scaffold; Go skeleton draws static 960x160 test frame in
   takeover (proves display path).
2. Chord + intercept toggle + OSD ON/OFF (proves input path, clean release).
3. `alsSource` -> render real arrangement (static). Then Remote Script
   `socketSource` for live data + playhead.
4. H/V zoom + scroll + follow playhead.
5. Jog scrub, transport, loop, locators commands + LEDs.
6. Hardening (crash-safe release, watcher, CPU), docs, release workflow.

## Verification
- `go vet`/unit tests for viewport math + render (golden image, ASCII-only
  assertion like `core/gfx/text` tests); protocol tests with a fake socket.
- On device (SSH deploy): toggle chord ON/OFF 20x, verify Live gets no button
  events while ON (Live Log.txt / Live UI unchanged), OSD text, zoom ranges,
  playhead sync while playing, kill -9 the hack -> Push recovers (filter off).
- `top` CPU of Live + hack during playback with 30+ tracks.
- `git log --format='%an <%ae>'` shows only Federico P..
