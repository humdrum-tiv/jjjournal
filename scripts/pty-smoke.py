#!/usr/bin/env python3
"""Launch a TUI binary in a pseudo-terminal and assert text on its last frame.

    scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --expect "revset:" \
        --wait 12 --key j --after 3 --expect "Commit ID"

Keys are sent in order after --wait seconds, each followed by --after seconds.
Exit status 0 when every --expect string appears on the final frame; otherwise
the cleaned frame is printed and the status is 1.
"""
import argparse
import fcntl
import os
import pty
import re
import select
import signal
import struct
import sys
import termios
import time

ANSI = re.compile(r"\x1b\[[0-9;?>=]*[ -/]*[@-~]|\x1b\][^\x07]*\x07|\x1b[=>]")


def pump(fd, seconds, out):
    end = time.time() + seconds
    while time.time() < end:
        ready, _, _ = select.select([fd], [], [], 0.2)
        if ready:
            try:
                out.append(os.read(fd, 65536))
            except OSError:
                return


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("binary")
    ap.add_argument("repo")
    ap.add_argument("--expect", action="append", default=[])
    ap.add_argument("--wait", type=float, default=8.0)
    ap.add_argument("--key", action="append", default=[])
    ap.add_argument("--after", type=float, default=3.0)
    ap.add_argument("--cols", type=int, default=200)
    ap.add_argument("--rows", type=int, default=45)
    args = ap.parse_args()
    args.binary = os.path.abspath(args.binary)

    pid, fd = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        os.chdir(args.repo)
        os.execvp(args.binary, [args.binary])
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", args.rows, args.cols, 0, 0))
    chunks = []
    pump(fd, args.wait, chunks)
    for key in args.key:
        os.write(fd, key.encode())
        pump(fd, args.after, chunks)
    try:
        os.write(fd, b"q")
        pump(fd, 0.7, chunks)
    except OSError:
        pass
    try:
        os.kill(pid, signal.SIGTERM)
    except ProcessLookupError:
        pass

    text = ANSI.sub("", b"".join(chunks).decode("utf-8", "replace")).replace("\r", "")
    lines = [l.rstrip() for l in text.split("\n") if l.strip()]
    frame = "\n".join(lines)
    missing = [e for e in args.expect if e not in frame]
    if missing:
        print(frame[-6000:])
        print("\nMISSING:", missing, file=sys.stderr)
        return 1
    print("ok:", ", ".join(args.expect))
    return 0


if __name__ == "__main__":
    sys.exit(main())
