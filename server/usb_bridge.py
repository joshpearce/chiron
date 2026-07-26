"""USB transport bridge: ferries exchanges between the iPad app and book-server
over the USB cable when there is no Wi-Fi link.

The app cannot initiate TCP over USB, so the direction inverts: the app parks
its exchange request in an outbox behind a tiny HTTP listener (port 8081);
this script polls it through usbmuxd and delivers the server's response back.

Prereqs:  brew install libimobiledevice   (provides iproxy)
Run:      .venv/bin/python usb_bridge.py
Leaves iproxy running as a child; Ctrl-C tears both down.
"""

import subprocess
import sys
import time

import httpx

IPROXY_PORT = 8081       # local port -> device port 8081
SERVER = "http://127.0.0.1:8080"
POLL_S = 2.0


def main():
    proc = subprocess.Popen(["iproxy", str(IPROXY_PORT), "8081"],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    print(f"iproxy up (pid {proc.pid}); polling iPad outbox every {POLL_S}s. Ctrl-C to stop.")
    try:
        while True:
            time.sleep(POLL_S)
            try:
                r = httpx.get(f"http://127.0.0.1:{IPROXY_PORT}/pending", timeout=3)
            except httpx.HTTPError:
                continue  # cable out or app backgrounded; keep polling
            if r.status_code != 200 or not r.content:
                continue
            print("exchange request received over USB; forwarding to book-server...")
            try:
                resp = httpx.post(f"{SERVER}/exchange", content=r.content,
                                  headers={"Content-Type": "application/json"},
                                  timeout=600)
                resp.raise_for_status()
            except httpx.HTTPError as e:
                print(f"  book-server error: {e}", file=sys.stderr)
                continue
            try:
                httpx.post(f"http://127.0.0.1:{IPROXY_PORT}/deliver",
                           content=resp.content,
                           headers={"Content-Type": "application/json"}, timeout=10)
                print("  delivered response to iPad.")
            except httpx.HTTPError as e:
                print(f"  delivery failed (cable pulled?): {e}", file=sys.stderr)
    except KeyboardInterrupt:
        pass
    finally:
        proc.terminate()


if __name__ == "__main__":
    main()
