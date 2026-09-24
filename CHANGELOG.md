# Changelog

## [Unreleased]

- Project created. Plan added.
- Milestone 1: Go skeleton draws a static 960x160 test frame in display takeover; `-preview` writes a PNG without a device.
- Milestone 2: Shift+Session (CC49+CC51) toggles Arrangement Mode: display takeover + MIDI intercept, ON/OFF message on screen, release on exit. `scripts/deploy.sh` runs it on the Push.
- `core` now taken from GitHub (core v0.2.0), no local replace.
- Toggle changed: Shift+Session shows the view (title "Arrangement Mode"); the second press closes it at once, no message.
- Milestone 3: PushHackArrangement Remote Script streams the open set (tracks, clips, exact colours, locators, loop, time signature, playhead) over a Unix socket. The hack draws it fit to the screen, with a moving playhead. The saved-file (.als) reader is now only for `make preview` and `-set`.
- All button LEDs and the pad grid go dark while Arrangement Mode is on.
- Removed the "Arrangement Mode" title from the view.
