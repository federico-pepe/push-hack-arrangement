import json
import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "remote-script"))
from snapshot import build_snapshot, pos_json  # noqa: E402
from commands import apply_commands  # noqa: E402


class O:
    def __init__(self, **kw):
        self.__dict__.update(kw)


def clip(s, e, name, color):
    return O(start_time=s, end_time=e, name=name, color=color)


class SnapshotTest(unittest.TestCase):
    def song(self):
        grp = O(name="Drums", is_foldable=True, has_midi_input=False, color=0xFF0000,
                group_track=None)
        kick = O(name="Kick", is_foldable=False, has_midi_input=True, color=0x00FF00,
                 group_track=grp, arrangement_clips=[clip(0, 8, "A", 0x112233), clip(8, 16, "B", 0x445566)])
        aud = O(name="Loop", is_foldable=False, has_midi_input=False, color=0x0000FF,
                group_track=None, arrangement_clips=[])
        return O(tracks=[grp, kick, aud], cue_points=[O(time=4.0, name="Drop")],
                 loop=True, loop_start=8.0, loop_length=16.0,
                 signature_numerator=3, signature_denominator=4, tempo=124.0,
                 current_song_time=2.5, is_playing=True)

    def test_snapshot(self):
        s = build_snapshot(self.song())
        self.assertEqual([t["kind"] for t in s["tracks"]], ["group", "midi", "audio"])
        self.assertEqual(s["tracks"][1]["group"], 0)
        self.assertEqual(s["tracks"][2]["group"], -1)
        self.assertEqual(s["tracks"][1]["clips"][0], [0.0, 8.0, "A", 0x112233])
        self.assertEqual(s["tracks"][0]["clips"], [])
        self.assertEqual(s["locators"], [{"time": 4.0, "name": "Drop"}])
        self.assertEqual(s["loop"], {"on": True, "start": 8.0, "length": 16.0})
        self.assertEqual(s["sig"], [3, 4])
        json.dumps(s)

    def test_pos(self):
        p = json.loads(pos_json(self.song()))
        self.assertEqual(p, {"t": "pos", "time": 2.5, "playing": True, "bta": False})

    def test_bad_track_does_not_break(self):
        song = self.song()
        song.tracks.append(O())  # no attributes at all
        s = build_snapshot(song)
        self.assertEqual(len(s["tracks"]), 4)


class CommandsTest(unittest.TestCase):
    def song(self):
        calls = []

        class S:
            current_song_time = 0.0
            is_playing = False
            back_to_arranger = 1
            tracks = []

            def start_playing(self):
                calls.append("start")
                self.is_playing = True

            def stop_playing(self):
                calls.append("stop")
                self.is_playing = False
        return S(), calls

    def test_only_last_set_time_and_others(self):
        s, calls = self.song()
        n = apply_commands(s, [{"t": "set_time", "v": 1}, {"t": "set_time", "v": 9.5},
                               {"t": "play_toggle"}, {"t": "bta"}, {"t": "bogus"}])
        self.assertEqual(n, 3)
        self.assertEqual(s.current_song_time, 9.5)
        self.assertEqual(calls, ["start"])
        self.assertEqual(s.back_to_arranger, 0)

    def test_toggle_stops(self):
        s, calls = self.song()
        s.is_playing = True
        apply_commands(s, [{"t": "play_toggle"}])
        self.assertEqual(calls, ["stop"])

    def test_negative_time_clamped(self):
        s, _ = self.song()
        apply_commands(s, [{"t": "set_time", "v": -4}])
        self.assertEqual(s.current_song_time, 0.0)


if __name__ == "__main__":
    unittest.main()
