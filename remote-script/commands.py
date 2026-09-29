# Commands from the hack to Live. No Live imports: takes a Song-like object.
#   {"t":"set_time","v":<beats>}                  move the playhead / insert marker
#   {"t":"play_toggle"}                           start playing, or stop if playing
#   {"t":"bta"}                                   Back to Arrangement
#   {"t":"loop_toggle"}                           flip the loop on/off flag
#   {"t":"punch_in_toggle"}                       flip Punch In
#   {"t":"punch_out_toggle"}                      flip Punch Out
#   {"t":"set_loop","start":<beats>,"length":<beats>}   move/resize the loop

# Commands whose value only ever matters as "the latest one": several may
# arrive in one tick (jog wheel, loop knobs), and applying every one in order
# is wasted work -- only the last matters.
COALESCED = ("set_time", "set_loop")


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
    elif kind == "loop_toggle":
        song.loop = not song.loop
    elif kind == "punch_in_toggle":
        song.punch_in = not song.punch_in
    elif kind == "punch_out_toggle":
        song.punch_out = not song.punch_out
    elif kind == "set_loop":
        song.loop_start = max(0.0, float(msg.get("start", song.loop_start)))
        song.loop_length = max(0.0001, float(msg.get("length", song.loop_length)))
    else:
        return False
    return True


def apply_commands(song, msgs):
    """Apply a batch. Coalesced kinds (see COALESCED) only apply their LAST
    message; everything else applies in order, first."""
    latest = {}
    rest = []
    for m in msgs:
        kind = m.get("t")
        if kind in COALESCED:
            latest[kind] = m
        else:
            rest.append(m)
    n = 0
    for m in rest:
        if apply_command(song, m):
            n += 1
    for m in latest.values():
        if apply_command(song, m):
            n += 1
    return n
