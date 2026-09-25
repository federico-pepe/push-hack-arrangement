# Push Hack: Arrangement

Shows the Live Arrangement on the Push 3 screen. Zoom in time and in track height.
Press **Shift + Session** to turn Arrangement Mode on or off. The view shows
the text "Arrangement Mode". The second press closes the view at once.

While the mode is on, the hack uses push-manager's MIDI intercept. Live does not
see the Push buttons, pads and encoders.

**Status:** milestone 5 (first part). Playhead control, transport and Back to Arrangement work.
See [plans/2026-09-24-arrangement-hack.md](plans/2026-09-24-arrangement-hack.md).

## Controls (while Arrangement Mode is on)

| Control | Action |
|---|---|
| Shift + Session | Turn the mode on or off |
| Jog wheel | Move the playhead. The step follows the zoom (from 1/64 beat to 32 beats). Turn fast to move more |
| Shift + jog wheel | Scroll the view in time |
| Play | Start or stop Live. The button is green while playing |
| Session | Back to Arrangement. The button lights bright white when Session clips override the arrangement. The clips on screen are dimmed until you press it |
| Volume dial | Zoom in time (horizontal) |
| Tempo dial | Zoom tracks (vertical). When lanes are tall enough, track and clip names appear |
| D-pad up / down | Scroll tracks. Hold to repeat |
| D-pad left / right | Scroll a quarter screen in time. Hold to repeat |

Every other LED goes dark while the mode is on.

While the song plays, the view scrolls so the playhead stays near the left. It stops following for 2 seconds after you move the view by hand.

The view shows
the text "Arrangement Mode". The second press closes the view at once.

While the mode is on, the hack uses push-manager's MIDI intercept. Live does not
see the Push buttons, pads and encoders.

**Status:** milestone 3. A Remote Script inside Live sends the arrangement of the **open** set
(also unsaved edits) and the playhead to the hack. The whole song fits on the screen. Zoom and
scroll come next. See [plans/2026-09-24-arrangement-hack.md](plans/2026-09-24-arrangement-hack.md).

The view shows one lane per track, clips in their colours, locators (blue lines), the loop (yellow bar
on the ruler) and the playhead (white line). When the mode turns on, all button LEDs and the pad
grid go dark.

## Requirements

- push-manager and push-display installed and running on the Push.
- The PushHackArrangement Remote Script. It is in `remote-script/` and `scripts/deploy.sh` copies it to
  Live's User Library. Enable it once: Live Preferences, a free control-surface slot, select
  `PushHackArrangement`, Input and Output = None. Then restart Live. Live's `Log.txt` should show
  "PushHackArrangement alive".

## Try it

```bash
scripts/deploy.sh            # builds, copies to push.local, runs in the foreground
```

Press Shift + Session on the Push. Press it again to switch off. Ctrl+C stops the
hack and gives the screen and MIDI back.

The hack talks to the Remote Script over `/tmp/push-hack-arrangement.sock`. Python tests: `make pytest`.

Without a device, `make preview` writes `build/preview.png` from `testdata/p3.als` (not in git; use your own set).


## Performance and safety

- With a 40-track, 342-clip set, drawing at about 8 frames per second used about 6% of one core in the hack and 4% in push-manager. Memory: 6 MB for the supervisor and about 12 MB for the working process.
- If the hack is killed while the mode is on, the supervisor gives the display and MIDI back and restarts the hack.

## Roadmap

Not done yet:

- Loop: toggle and set the loop. Locators: add and jump. Punch in/out. Record. Metronome.
- Jog step buttons (finer and coarser) and the jog-click play-start marker from the reference script.
- Editing: nudge, resize, split, duplicate, spacer. These are from the reference script and are outside v1.
- Send only the visible window for a very huge set (the Remote Script already slows down snapshots on big sets).
- Install through the catalog (needs a `catalog.json` entry and a new release). Then the boot service is created for you.
- Docs: `docs/architecture.md`, `docs/protocol.md`.
