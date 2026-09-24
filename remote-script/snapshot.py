# Build the arrangement snapshot from a Live Song object.
# No Live imports here, so it can be tested with fake objects.

import json


def _safe(fn, default=None):
    try:
        return fn()
    except Exception:
        return default


def build_snapshot(song):
    all_tracks = list(song.tracks)
    tracks = []
    for i, t in enumerate(all_tracks):
        if _safe(lambda: t.is_foldable, False):
            kind = "group"
        elif _safe(lambda: t.has_midi_input, False):
            kind = "midi"
        else:
            kind = "audio"
        group = -1
        g = _safe(lambda: t.group_track)
        if g is not None:
            group = _safe(lambda: all_tracks.index(g), -1)
        clips = []
        if kind != "group":
            for c in _safe(lambda: list(t.arrangement_clips), []):
                clips.append([
                    round(float(c.start_time), 4),
                    round(float(c.end_time), 4),
                    _safe(lambda: c.name, "") or "",
                    _safe(lambda: int(c.color), 0),
                ])
        tracks.append({
            "name": _safe(lambda: t.name, "") or "",
            "kind": kind,
            "color": _safe(lambda: int(t.color), 0),
            "group": group,
            "clips": clips,
        })
    locators = []
    for cp in _safe(lambda: list(song.cue_points), []):
        locators.append({"time": round(float(cp.time), 4), "name": _safe(lambda: cp.name, "") or ""})
    return {
        "t": "snapshot",
        "tracks": tracks,
        "locators": locators,
        "loop": {
            "on": bool(_safe(lambda: song.loop, False)),
            "start": float(_safe(lambda: song.loop_start, 0.0)),
            "length": float(_safe(lambda: song.loop_length, 0.0)),
        },
        "sig": [_safe(lambda: int(song.signature_numerator), 4),
                _safe(lambda: int(song.signature_denominator), 4)],
        "tempo": float(_safe(lambda: song.tempo, 120.0)),
    }


def snapshot_json(song):
    return json.dumps(build_snapshot(song), separators=(",", ":"), sort_keys=True)


def pos_json(song):
    return json.dumps({"t": "pos",
                       "time": round(float(song.current_song_time), 3),
                       "playing": bool(song.is_playing)}, separators=(",", ":"))
