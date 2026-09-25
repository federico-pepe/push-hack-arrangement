# Touch layer (planned)

**Status:** not built yet. This page is the design, so we can build it when we add more controls.

## Idea

The screen is small. We do not want a top bar and a bottom bar on it all the time.
Instead, the arrangement uses the whole screen. When you touch a knob or the jog wheel,
a temporary layer appears on top of the arrangement and shows what that control does
and its value. The layer goes away a moment after you stop touching.

## Input: touch notes

The knobs above the screen and the jog wheel are capacitive. They send a Note On
(velocity 127) when you touch them and a Note Off (or velocity 0) when you let go.
The hack already receives these events from the Push port, also while the MIDI
intercept hides them from Live. The current code ignores notes.

| Control | Touch note |
|---|---|
| Knobs 1 to 8 (above the screen) | 0 to 7 |
| Volume knob | 8 |
| Tempo knob | 10 |
| Jog wheel | 11 |

Other touch notes (not planned for the layer): touch strip 12, D-pad center 13.
Source: `docs/push3-button-map.md` in the ableton-push-hack repo.

## Behavior

1. Note On of a control: show the layer for that control at once.
2. Note Off: start a timer of about 500 ms (one constant, easy to change).
3. When the timer ends, remove the layer.
4. A new touch during the timer cancels the timer. The layer stays.
5. If two controls are touched at once, the layer stays until both are released and the timer ends.
6. Turning the mode off removes the layer and stops the timer.
7. A short message such as "ZOOM: TIME" (already in the code) can move into this layer.

## Content per control (to decide)

Start with one row per control. Suggested first version:

| Control | Layer shows |
|---|---|
| Jog wheel | Playhead position (bar, beat), grid step (for example 1/16) |
| Volume knob | View start and end (bars) |
| Tempo knob | Zoom mode (time or tracks) and the zoom level |
| Knobs 1 to 8 | Label and value, once they have a job |

Rule for later work: **when you add a control, add its layer content in the same change,
and add a row to this table and to the controls table in the README.**

## Drawing

- The layer is drawn into the same 960x160 image, after the arrangement, so it needs no new display code.
- Use a semi-transparent dark bar (top, bottom or both) with ASCII text only. The panel font has no other glyphs.
- Open choices: top bar, bottom bar or both, and a plain cut or a short fade.
- A frame is redrawn only when something changes. The timer end must also request one redraw.

## Implementation sketch

- `midi.go`: also handle `alsaseq.EvNoteOn` and `alsaseq.EvNoteOff` from the Push port (channel 0). Pass `(note, on)` to a new `onTouch` callback.
- New `touch.go`: holds the set of touched notes, the timer, and the current layer. It offers `active() (layer, bool)` and calls `changed()` on every change.
- `view.go` or `render.go`: `viewport` gets a `layer` field. `renderArrangement` draws it last, like `drawToast`.
- Tests: touch on, off, timer end, new touch during the timer, two controls at once, mode off.
- Do not run the layer when the mode is off. Reset it in `modeCtl.toggle`.
