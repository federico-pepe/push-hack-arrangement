# Push Hack: Arrangement

Shows the Live Arrangement on the Push 3 screen. Zoom in time and in track height.
Press Shift + Session to turn Arrangement Mode on or off. The screen shows
"Arrangement Mode: ON" or "Arrangement Mode: OFF".

The hack uses push-manager's intercept mode. While the mode is on, Live does not
see the Push buttons, pads and encoders.

**Status:** milestone 1 (static test frame). `make preview` writes `build/preview.png`; `./arrangement` on the Push shows it in takeover. See [plans/2026-09-24-arrangement-hack.md](plans/2026-09-24-arrangement-hack.md).

## Requirements

- push-manager and push-display installed and running on the Push.
