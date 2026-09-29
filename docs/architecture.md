# Architecture

The hack has two parts.

## 1. Remote Script (`remote-script/`, runs inside Live)

`PushHackArrangement` is a Live MIDI Remote Script. It has no controls and no MIDI mapping.
Every 100 ms, Live calls `update_display`. The script then does four things:

1. It accepts a connection on the Unix socket `/tmp/push-hack-arrangement.sock`.
2. It reads command lines from the hack and applies them (`commands.py`).
3. About once per second it builds a snapshot of the open set (`snapshot.py`): tracks, clips, locators, time signature, tempo. It sends the snapshot only when it changed.
4. It sends the playhead, transport and loop/punch state (`pos`) whenever any of it changed -- every tick, not once a second, so a loop toggle or a knob turn shows up almost at once.

`snapshot.py` and `commands.py` do not import Live. This lets the tests in `tests/` run on any computer.

## 2. Go program (`src/`, runs as a normal process)

| File | Job |
|---|---|
| `main.go` | Start-up, wiring, signal handling. Releases display and MIDI on exit |
| `supervisor.go` | The started process is a parent. It runs the hack as a child and restarts it. If the child dies with the mode on (state file `/tmp/push-hack-arrangement.mode`), it releases display and MIDI |
| `bench.go` | `-bench` mode to measure CPU on the device |
| `midi.go` | Own ALSA port, subscribed to Push 3 (client 16, port 0). Reads every button and dial |
| `chord.go` | Shift (CC49) + Session (CC51) chord, 500 ms debounce |
| `mode.go` | Arrangement Mode on/off. ON = display takeover + MIDI intercept. OFF = give both back at once |
| `controls.go` | Jog wheel (scrub cursor; Shift = real seek), jog press (play-start anchor), Play, Session (Back to Arrangement), Page </> (locator nav), the 3 loop/punch top buttons, loop start/length encoders, Volume knob (scroll), Tempo knob (zoom, click to switch), D-pad with repeat |
| `view.go` | Zoom and scroll state, playhead follow. Horizontal zoom always pivots on the playhead (or the scrub cursor, if one is active). Tracks whether each D-pad direction can still scroll further (`computeAvailability`) and tells `leds.go`'s `syncDPadLEDs` via a callback (`onScrollAvailability`) after every zoom/scroll. All values are clamped for the current set |
| `touch.go` | The touch layer: knob/jog-wheel touch shows a status bar over the arrangement. See `docs/touch-layer.md` |
| `bbs.go` | Bars.beats.16th formatting (`bbsPosition`, `bbsLength`), matching Live's own display |
| `render.go` | Draws the 960x160 image: ruler, lanes, clips, names, locators, loop (region bar + on/off boundary lines), play-start anchor (triangle), scrub cursor, playhead |
| `source.go` | Connection to the Remote Script, the current set, commands to Live. Also holds local-only state Live never reports back: the play-start anchor, the selected locator, and the scrub cursor |
| `model.go` | Data types (Set, Track, Clip, Locator, Loop) |
| `leds.go` | LED blackout and the few LEDs we light (Play, Session, Page </>, the 3 loop/punch buttons, the 4 D-pad directions) |
| `depcheck.go` | Logs when push-manager or push-display is missing |
| `als.go`, `colors.go` | Reader for a saved `.als` file. Only for `make preview` and `-set`. Not used on the Push |

## How the hack takes over the Push (intercept mode)

- The hack never writes to the display memory. It sends images to push-manager (`pmclient.PushImage`).
- While the mode is on, `POST /api/midi/filter` makes push-display hide button, pad and dial MIDI from Live.
- The hack still receives that MIDI, because it has its own subscription to the Push port.
- LEDs are written straight to the Push port with our own ALSA client.

## Rules the code must keep

- Every exit path releases the display and the MIDI intercept (`modeCtl.release`).
- No access to `/dev/snd` before `alsaseq.WaitForBootSettle()` (30 s after cold boot).
- Text on the screen is ASCII only.
- The boot service starts the hack as `arrangement -config <hack.json>`. Keep accepting `-config`.
- Redraws are limited to about 8 per second (`minPushGap`). Each one is a PNG encode and an HTTP post.
