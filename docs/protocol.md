# Protocol: Remote Script and hack

One JSON object per line. Only the hack connects. Where the Remote Script listens depends on the system (`remote-script/transport.py`):

| System | Endpoint |
|---|---|
| macOS, Linux | Unix socket `/tmp/push-hack-arrangement.sock`, owner only |
| Windows | TCP `127.0.0.1:7720`. Python on Windows has no Unix sockets |

The messages are the same on both. A client uses the endpoint of its own system.

## Live to hack

`snapshot`: sent on connect and when the arrangement changes (about once a second at most).

```json
{"t":"snapshot",
 "tracks":[{"name":"Kick","kind":"midi","color":16711680,"group":0,
            "clips":[[0.0, 8.0, "Clip name", 5539044]]}],
 "locators":[{"time":4.0,"name":"Drop"}],
 "sig":[4,4], "tempo":124.0}
```

- `kind`: `midi`, `audio` or `group`.
- `group`: index of the parent group track, or -1.
- A clip is `[start, end, name, color]`. Times are in beats. Colors are `0xRRGGBB` as a number.
- Return tracks are not sent.
- No loop or punch state here -- see `pos` below. A snapshot is built at most once a
  second; `pos` runs every tick, so a loop toggle shows up almost at once instead of
  waiting for the next snapshot.

`pos`: sent whenever any of these values changes (up to once per Live tick, ~100ms).

```json
{"t":"pos","time":12.5,"playing":true,"bta":false,
 "loop_on":true,"loop_start":8.0,"loop_length":16.0,
 "punch_in":false,"punch_out":false}
```

- `bta` is true when Session clips override the arrangement (the Back to Arrangement button is lit in Live).
- `loop_on`/`loop_start`/`loop_length` mirror `song.loop`/`loop_start`/`loop_length`.
- `punch_in`/`punch_out` mirror `song.punch_in`/`punch_out`.

## Hack to Live

| Command | Effect |
|---|---|
| `{"t":"set_time","v":<beats>}` | Move the playhead (a real seek -- see below) |
| `{"t":"play_toggle"}` | Start playing, or stop if playing |
| `{"t":"bta"}` | Back to Arrangement (`song.back_to_arranger = 0`) |
| `{"t":"loop_toggle"}` | Flip `song.loop` |
| `{"t":"punch_in_toggle"}` | Flip `song.punch_in` |
| `{"t":"punch_out_toggle"}` | Flip `song.punch_out` |
| `{"t":"set_loop","start":<beats>,"length":<beats>}` | Move/resize the loop |

`set_time` and `set_loop` are coalesced (`commands.py`'s `COALESCED`): if several of
the same kind arrive in one tick (the jog wheel or a loop knob can send many), only
the last one is applied. Every other command applies in the order it arrived.

Unknown commands are ignored.

## What does NOT go over this socket

The hack's plain jog wheel (no Shift) moves a "scrub cursor" -- purely local state,
so you can look around the timeline without an audible jump in Live's real playback.
Nothing is sent for that. Only Shift+jog, Page </> (locator jump) and the Play-anchor
jump send `set_time` -- see `controls.go`'s doc comment and `docs/architecture.md`.
