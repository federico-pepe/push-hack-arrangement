# Push Hack: Arrangement

Shows the Live Arrangement on the Push 3 screen. Zoom in time and in track height.
Press **Shift + Session** to turn Arrangement Mode on or off. The screen shows
"Arrangement Mode: ON" or "Arrangement Mode: OFF".

While the mode is on, the hack uses push-manager's MIDI intercept. Live does not
see the Push buttons, pads and encoders.

**Status:** milestone 2. The toggle, the message and the intercept work. The screen
shows a fixed test pattern. Real arrangement data comes in a later milestone.
See [plans/2026-09-24-arrangement-hack.md](plans/2026-09-24-arrangement-hack.md).

## Requirements

- push-manager and push-display installed and running on the Push.

## Try it

```bash
scripts/deploy.sh            # builds, copies to push.local, runs in the foreground
```

Press Shift + Session on the Push. Press it again to switch off. Ctrl+C stops the
hack and gives the screen and MIDI back.

Without a device, `make preview` writes `build/preview.png`.

## Known issue to check on the device

The first Shift + Session press may also reach Live, because the intercept starts
only after the hack sees the chord.
