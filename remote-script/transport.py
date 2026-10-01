# Where the Remote Script listens. No Live imports, so tests can load it.
#
# macOS and Linux: a Unix socket, readable by the owner only. This is what the
# standalone hack and the PTA module use by default.
# Windows: Python has no AF_UNIX there, so a TCP port on 127.0.0.1 instead.
#
# One protocol on both: one JSON object per line (docs/protocol.md).

import os
import socket

UNIX_PATH = "/tmp/push-hack-arrangement.sock"
TCP_HOST = "127.0.0.1"
TCP_PORT = 7720  # outside 7701-7710, which the Push hack framework reserves


def endpoint(has_unix=None):
    """('unix', path) or ('tcp', (host, port)). has_unix is for tests."""
    if has_unix is None:
        has_unix = hasattr(socket, "AF_UNIX")
    if has_unix:
        return ("unix", UNIX_PATH)
    return ("tcp", (TCP_HOST, TCP_PORT))


def open_listener(ep=None):
    """Open a non-blocking listening socket. Returns (socket, endpoint)."""
    ep = ep or endpoint()
    kind, addr = ep
    if kind == "unix":
        if os.path.exists(addr):
            os.unlink(addr)
        s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        s.bind(addr)
        os.chmod(addr, 0o600)
    else:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        # SO_REUSEADDR on Windows lets another process take the port.
        if hasattr(socket, "SO_EXCLUSIVEADDRUSE"):
            s.setsockopt(socket.SOL_SOCKET, socket.SO_EXCLUSIVEADDRUSE, 1)
        else:
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(addr)
    s.listen(2)
    s.setblocking(False)
    return s, ep


def close_listener(s, ep):
    try:
        s.close()
    finally:
        if ep[0] == "unix" and os.path.exists(ep[1]):
            os.unlink(ep[1])
