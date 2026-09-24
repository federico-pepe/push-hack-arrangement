# Push Hack: Arrangement

Shows the Live Arrangement on the Push 3 screen. Zoom in time and in track height.
Press **Shift + Session** to turn Arrangement Mode on or off. The view shows
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

