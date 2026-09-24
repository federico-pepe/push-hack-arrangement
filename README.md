# Push Hack: Arrangement

Shows the Live Arrangement on the Push 3 screen. Zoom in time and in track height.
Press **Shift + Session** to turn Arrangement Mode on or off. The view shows
the text "Arrangement Mode". The second press closes the view at once.

While the mode is on, the hack uses push-manager's MIDI intercept. Live does not
see the Push buttons, pads and encoders.

**Status:** milestone 3a. Shift + Session shows the real arrangement from the newest saved
Live Set (`/data/Music/Ableton/Sets`). The whole song fits on the screen. Zoom, scroll and live
updates come next. See [plans/2026-09-24-arrangement-hack.md](plans/2026-09-24-arrangement-hack.md).

The view shows one lane per track, clips in their colours, locators (blue lines), the loop (yellow bar on
the ruler) and the saved insert marker (white line). It reads the saved file, so it changes only after
you save the set in Live. The reader uses an approximate colour table for Live's 70 colours.

## Requirements

- push-manager and push-display installed and running on the Push.

## Try it

```bash
scripts/deploy.sh            # builds, copies to push.local, runs in the foreground
```

Press Shift + Session on the Push. Press it again to switch off. Ctrl+C stops the
hack and gives the screen and MIDI back.

Without a device, `make preview` writes `build/preview.png` from `testdata/p3.als` (not in git; use your own set).

