# Protocol: Remote Script and hack

Unix socket `/tmp/push-hack-arrangement.sock`. One JSON object per line. Only the hack connects.

## Live to hack

`snapshot`: sent on connect and when the arrangement changes.

```json
{"t":"snapshot",
 "tracks":[{"name":"Kick","kind":"midi","color":16711680,"group":0,
            "clips":[[0.0, 8.0, "Clip name", 5539044]]}],
 "locators":[{"time":4.0,"name":"Drop"}],
 "loop":{"on":true,"start":8.0,"length":16.0},
 "sig":[4,4], "tempo":124.0}
```

- `kind`: `midi`, `audio` or `group`.
- `group`: index of the parent group track, or -1.
- A clip is `[start, end, name, color]`. Times are in beats. Colors are `0xRRGGBB` as a number.
- Return tracks are not sent.

`pos`: sent when a value changes.

```json
{"t":"pos","time":12.5,"playing":true,"bta":false}
```

`bta` is true when Session clips override the arrangement (the Back to Arrangement button is lit in Live).

## Hack to Live

| Command | Effect |
|---|---|
| `{"t":"set_time","v":<beats>}` | Move the playhead. If several arrive in one tick, only the last one counts |
| `{"t":"play_toggle"}` | Start playing, or stop if playing |
| `{"t":"bta"}` | Back to Arrangement (`song.back_to_arranger = 0`) |

Unknown commands are ignored.
