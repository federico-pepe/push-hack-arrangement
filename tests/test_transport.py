import os
import socket
import sys
import tempfile
import time
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "remote-script"))
import transport  # noqa: E402


def accept(srv, timeout=2.0):
    end = time.time() + timeout
    while time.time() < end:
        try:
            return srv.accept()[0]
        except (BlockingIOError, InterruptedError):
            time.sleep(0.01)
    raise AssertionError("no client accepted")


class EndpointTest(unittest.TestCase):
    def test_unix_where_available(self):
        self.assertEqual(transport.endpoint(has_unix=True), ("unix", "/tmp/push-hack-arrangement.sock"))

    def test_tcp_on_loopback_without_unix(self):
        kind, (host, port) = transport.endpoint(has_unix=False)
        self.assertEqual((kind, host, port), ("tcp", "127.0.0.1", 7720))
        self.assertFalse(1 <= 7701 <= port <= 7710, "port must stay outside the framework range")


class ListenerTest(unittest.TestCase):
    def roundtrip(self, srv, connect):
        client = connect()
        conn = accept(srv)
        client.sendall(b'{"t":"play_toggle"}\n')
        conn.setblocking(True)
        conn.settimeout(2)
        self.assertEqual(conn.recv(100), b'{"t":"play_toggle"}\n')
        conn.sendall(b'{"t":"pos"}\n')
        client.settimeout(2)
        self.assertEqual(client.recv(100), b'{"t":"pos"}\n')
        client.close()
        conn.close()

    def test_tcp(self):
        srv, ep = transport.open_listener(("tcp", ("127.0.0.1", 0)))
        try:
            port = srv.getsockname()[1]
            self.assertEqual(srv.getsockname()[0], "127.0.0.1")
            self.roundtrip(srv, lambda: socket.create_connection(("127.0.0.1", port), 2))
        finally:
            transport.close_listener(srv, ep)

    @unittest.skipUnless(hasattr(socket, "AF_UNIX"), "no unix sockets here")
    def test_unix_is_owner_only_and_removed_on_close(self):
        d = tempfile.mkdtemp(prefix="pha")
        path = os.path.join(d, "t.sock")
        srv, ep = transport.open_listener(("unix", path))
        try:
            self.assertEqual(os.stat(path).st_mode & 0o777, 0o600)

            def connect():
                c = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
                c.connect(path)
                return c
            self.roundtrip(srv, connect)
        finally:
            transport.close_listener(srv, ep)
        self.assertFalse(os.path.exists(path))
        os.rmdir(d)

    @unittest.skipUnless(hasattr(socket, "AF_UNIX"), "no unix sockets here")
    def test_unix_replaces_a_stale_socket_file(self):
        d = tempfile.mkdtemp(prefix="pha")
        path = os.path.join(d, "t.sock")
        open(path, "w").close()  # left behind by a crashed Live
        srv, ep = transport.open_listener(("unix", path))
        transport.close_listener(srv, ep)
        os.rmdir(d)


if __name__ == "__main__":
    unittest.main()
