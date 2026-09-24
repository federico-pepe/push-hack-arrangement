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
