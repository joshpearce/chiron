"""Prove the USB transport without the iPad.

Stands up a mock device listener that behaves exactly like the app's embedded
Network.framework listener (GET /pending serves a parked exchange request,
POST /deliver accepts the response), runs the real usb_bridge against it, and
asserts a complete round trip: request out, chapter back.

Run:  .venv/bin/python test_usb_bridge.py
Requires book-server on :8080.
"""

from __future__ import annotations

import json
import subprocess
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

import httpx

HERE = Path(__file__).parent
DEVICE_PORT = 8099
SERVER = "http://127.0.0.1:8080"

REQUEST = json.dumps({"subject": "ai", "phase": "start"}).encode()
delivered: dict = {}
served = threading.Event()


class MockDevice(BaseHTTPRequestHandler):
    """Mimics Sync.swift's listener: /ping, /pending, /deliver."""

    def _respond(self, code: int, body: bytes | None):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body) if body else 0))
        self.end_headers()
        if body:
            self.wfile.write(body)

    def do_GET(self):
        if self.path == "/ping":
            self._respond(200, b'{"ok":true}')
        elif self.path == "/pending":
            if delivered:                      # outbox already drained
                self._respond(204, None)
            else:
                served.set()
                self._respond(200, REQUEST)
        else:
            self._respond(404, None)

    def do_POST(self):
        if self.path == "/deliver":
            n = int(self.headers.get("Content-Length", 0))
            delivered["body"] = self.rfile.read(n)
            self._respond(200, b'{"ok":true}')
        else:
            self._respond(404, None)

    def log_message(self, *args):
        pass                                    # keep test output clean


def main() -> int:
    try:
        httpx.get(f"{SERVER}/health", timeout=3).raise_for_status()
    except httpx.HTTPError:
        print("FAIL book-server not reachable on :8080")
        return 1

    httpd = HTTPServer(("127.0.0.1", DEVICE_PORT), MockDevice)
    threading.Thread(target=httpd.serve_forever, daemon=True).start()
    print(f"mock device listening on :{DEVICE_PORT}")

    bridge = subprocess.Popen(
        [sys.executable, str(HERE / "usb_bridge.py")],
        env={**__import__("os").environ,
             "CHIRON_NO_IPROXY": "1",
             "CHIRON_DEVICE_PORT": str(DEVICE_PORT),
             "CHIRON_POLL_S": "1.0"},
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)

    try:
        deadline = time.time() + 300           # chapter generation is slow
        while time.time() < deadline and "body" not in delivered:
            if bridge.poll() is not None:
                print("FAIL bridge exited early:", bridge.stdout.read()[-500:])
                return 1
            time.sleep(1)
    finally:
        bridge.terminate()

    if "body" not in delivered:
        print("FAIL bridge never delivered a response"
              f" (device served request: {served.is_set()})")
        return 1

    payload = json.loads(delivered["body"])
    chapter = payload.get("chapter")
    if not chapter:
        print("FAIL delivered payload carried no chapter:", list(payload))
        return 1
    print(f"PASS round trip: unit={chapter['unit']} "
          f"html={len(chapter['html'])} chars, {len(chapter['check'])} check items")
    return 0


if __name__ == "__main__":
    sys.exit(main())
