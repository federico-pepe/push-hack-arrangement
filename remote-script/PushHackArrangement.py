# PushHackArrangement: read-only bridge. Streams the open Live Set's arrangement
# to the arrangement hack over a local Unix socket. No controls, no MIDI mapping.
#
# Live -> hack, one JSON per line:
#   {"t":"snapshot", tracks, locators, loop, sig, tempo}   on connect + when changed
#   {"t":"pos", time, playing}                             when changed (~10 Hz max)

import os
import socket

from _Framework.ControlSurface import ControlSurface

from .snapshot import snapshot_json, pos_json

SOCK_PATH = "/tmp/push-hack-arrangement.sock"
SNAPSHOT_EVERY_TICKS = 10        # update_display runs ~10x/s -> check ~1x/s
MAX_OUT_BYTES = 8 * 1024 * 1024  # drop a client that cannot keep up


class PushHackArrangement(ControlSurface):

    def __init__(self, c_instance):
        ControlSurface.__init__(self, c_instance)
        self._srv = None
        self._clients = []
        self._tick = 0
        self._last_snapshot = None
        self._last_pos = None
        self._open_server()
        self.log_message("PushHackArrangement alive (socket %s)" % SOCK_PATH)

    # ---- socket ----
    def _open_server(self):
        try:
            if os.path.exists(SOCK_PATH):
                os.unlink(SOCK_PATH)
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.bind(SOCK_PATH)
            os.chmod(SOCK_PATH, 0o600)
            s.listen(2)
            s.setblocking(False)
            self._srv = s
        except Exception as e:
            self.log_message("PushHackArrangement: cannot open socket: %s" % e)
            self._srv = None

    def _accept(self):
        if self._srv is None:
            return
        try:
            conn, _ = self._srv.accept()
        except (BlockingIOError, InterruptedError):
            return
        except Exception:
            return
        conn.setblocking(False)
        self._clients.append({"s": conn, "out": b""})
        self._last_snapshot = None   # force a snapshot for the new client
        self._last_pos = None

    def _drop(self, c):
        try:
            c["s"].close()
        except Exception:
            pass
        if c in self._clients:
            self._clients.remove(c)

    def _queue(self, line):
        data = (line + "\n").encode("utf-8")
        for c in list(self._clients):
            c["out"] += data
            if len(c["out"]) > MAX_OUT_BYTES:
                self._drop(c)

    def _flush(self):
        for c in list(self._clients):
            if not c["out"]:
                continue
            try:
                n = c["s"].send(c["out"])
                c["out"] = c["out"][n:]
            except (BlockingIOError, InterruptedError):
                pass
            except Exception:
                self._drop(c)

    def _read(self):
        # v1 has no commands: read only to notice a closed connection.
        for c in list(self._clients):
            try:
                if c["s"].recv(4096) == b"":
                    self._drop(c)
            except (BlockingIOError, InterruptedError):
                pass
            except Exception:
                self._drop(c)

    # ---- Live tick ----
    def update_display(self):
        ControlSurface.update_display(self)
        self._accept()
        self._read()
        if not self._clients:
            return
        self._tick += 1
        song = self.song()
        try:
            if self._last_snapshot is None or self._tick % SNAPSHOT_EVERY_TICKS == 0:
                snap = snapshot_json(song)
                if snap != self._last_snapshot:
                    self._last_snapshot = snap
                    self._queue(snap)
            pos = pos_json(song)
            if pos != self._last_pos:
                self._last_pos = pos
                self._queue(pos)
        except Exception as e:
            self.log_message("PushHackArrangement: update error: %s" % e)
        self._flush()

    def disconnect(self):
        for c in list(self._clients):
            self._drop(c)
        try:
            if self._srv is not None:
                self._srv.close()
            if os.path.exists(SOCK_PATH):
                os.unlink(SOCK_PATH)
        except Exception:
            pass
        ControlSurface.disconnect(self)
