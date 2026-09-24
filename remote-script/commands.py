# Commands from the hack to Live. No Live imports: takes a Song-like object.
#   {"t":"set_time","v":<beats>}   move the playhead / insert marker
#   {"t":"play_toggle"}            start playing, or stop if playing
#   {"t":"bta"}                    Back to Arrangement


def back_to_arrangement(song):
    # Same call the Push3ArrangementMode script uses: writing 0 triggers it.
    if hasattr(song, "back_to_arranger"):
        song.back_to_arranger = 0
        return
    for t in list(song.tracks):
        try:
            if hasattr(t, "back_to_arranger"):
                t.back_to_arranger = 0
        except Exception:
            pass


def apply_command(song, msg):
    kind = msg.get("t")
    if kind == "set_time":
        song.current_song_time = max(0.0, float(msg.get("v", 0.0)))
    elif kind == "play_toggle":
        if song.is_playing:
            song.stop_playing()
        else:
            song.start_playing()
    elif kind == "bta":
        back_to_arrangement(song)
    else:
        return False
    return True


def apply_commands(song, msgs):
    """Apply a batch. Only the LAST set_time counts (jog sends many)."""
    last_time = None
    rest = []
    for m in msgs:
        if m.get("t") == "set_time":
            last_time = m
        else:
            rest.append(m)
    n = 0
    if last_time is not None and apply_command(song, last_time):
        n += 1
    for m in rest:
        if apply_command(song, m):
            n += 1
    return n
