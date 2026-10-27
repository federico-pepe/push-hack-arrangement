# CLAUDE.md

Push-hack module for Ableton Push 3: Arrangement view on the Push screen. Go host.
It reads Live through the shared PushHack Remote Script (in `ableton-push-hack`). Follows the conventions of the other push-hack
repos (keyboard-visualizer, braids). Plan: `plans/`.

## Push family context

Shared facts for all Push repos (repo map, git identity, `core/` pinning, cross-repo hardware facts):

@~/.claude/push-family.md

## Rules
- Git identity in this repo: **Federico P.** `<6317270+federico-pepe@users.noreply.github.com>`. Never the @ableton account.
- `core` comes from GitHub (`github.com/federico-pepe/ableton-push-hack/core`, tag in `src/go.mod`). No local `replace`.
- Draw on screen only through push-manager (`pmclient`). Never mmap shm.
- Screen strings are ASCII only.
- Live data comes from the PushHack Remote Script (`remote-script/PushHack/` in `ableton-push-hack`), topics `arrangement` and `position`. This repo has no script of its own. A change to a topic goes into `ableton-push-hack` first. `docs/protocol.md` says which topics this hack uses.
- A test must never open `/tmp/push-hack.sock` on a computer where Live runs: the script deletes an existing socket file before it binds. Tests set `liveSocketPath` to their own path.
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
- Deploy by hand: `scripts/deploy.sh` (needs a terminal). Without one: stop the process, `scp` the binary, start it in the background.
- Docs: `docs/architecture.md`, `docs/protocol.md`, `docs/touch-layer.md` (planned touch layer). Plan and status: `plans/`.
- When you add a control: update the README controls table and the touch-layer table in `docs/touch-layer.md`.
