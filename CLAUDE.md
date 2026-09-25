# CLAUDE.md

Push-hack module for Ableton Push 3: Arrangement view on the Push screen. Go host
+ (later) a Live Remote Script. Follows the conventions of the other push-hack
repos (keyboard-visualizer, braids). Plan: `plans/`.

## Rules
- Git identity in this repo: **Federico P.** `<6317270+federico-pepe@users.noreply.github.com>`. Never the @ableton account.
- `core` comes from GitHub (`github.com/federico-pepe/ableton-push-hack/core`, tag in `src/go.mod`). No local `replace`.
- Draw on screen only through push-manager (`pmclient`). Never mmap shm.
- Screen strings are ASCII only.
- Any exit path must release display takeover and MIDI intercept (`modeCtl.release`).
- No `/dev/snd` access before `alsaseq.WaitForBootSettle()`.
- Code comments: caveman skill. Docs: simple-english skill. Update CHANGELOG `[Unreleased]` per notable change.

## Commands
`make test vet preview build` ; `scripts/deploy.sh [host]` runs it on the Push.
Release: bump `hack.json` version, tag `vX.Y.Z-alpha`, push tag (workflow builds + writes release.json).

## Lessons from the device
- The catalog boot service runs `<binary> -config <hack.json>`. The hack must accept `-config`.
- Push buttons: Session (CC51) has a greyscale LED, so use palette 120 for "on". Play is RGB (white 120, green 126).
- Live repaints LEDs after commands (Play, Back to Arrangement). `burstLEDs` repeats the blackout for about 1 s to hide it.
- Set the LED colour of Play at once on a press; do not wait for Live's answer.
- Deploy by hand: `scripts/deploy.sh` (needs a terminal). Without one: stop the process, `scp` the binary and `remote-script/*.py`, start it in the background. Restart Live after changing the Remote Script.
- Docs: `docs/architecture.md`, `docs/protocol.md`. Plan and status: `plans/`.
