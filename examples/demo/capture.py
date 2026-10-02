#!/usr/bin/env python3
"""Record live PTY output; render its observed timeline without speeding it up."""
import argparse
import codecs
import errno
import fcntl
import hashlib
import json
import os
from pathlib import Path
import pty
import re
import struct
import subprocess
import termios
import time

ROOT = Path(__file__).resolve().parents[2]
CAST = ROOT / 'docs-site/docs/assets/demo.cast'
GIF = CAST.with_suffix('.gif')
REPORT = ROOT / '.pose/reports/2026-10-02-demo-recording.json'


def validate():
    lines = [json.loads(line) for line in CAST.read_text().splitlines()]
    header, events = lines[0], lines[1:]
    report = json.loads(REPORT.read_text())
    assert header['version'] == 2 and header['width'] == 112 and header['height'] == 30
    assert events and all(e[1] == 'o' and isinstance(e[2], str) for e in events)
    assert all(a[0] <= b[0] for a, b in zip(events, events[1:]))
    assert 0 < report['seconds'] < 60 and report['exit_code'] == 0
    text = ''.join(e[2] for e in events)
    assert 'R1 has no trace entry' in text and 'spec.trace.missing=0' in text
    assert 'Done' in text and 'evidence' in text
    for path in [CAST, GIF]:
        assert hashlib.sha256(path.read_bytes()).hexdigest() == report['sha256'][path.name]
    print('PASS: live recording, ordered events, real block/resolution, source revision and asset digests')


def render(events, duration):
    # The scenario uses SGR styling and line output only, not a full-screen TUI.
    from PIL import Image, ImageDraw, ImageFont
    font = ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf', 15)
    frames = []
    text, index = '', 0
    ansi = re.compile(r'\x1b\[[0-9;]*[A-Za-z]')
    for frame in range(int(duration * 10) + 1):
        at = frame / 10
        while index < len(events) and events[index][0] <= at:
            text += events[index][2]
            index += 1
        cleaned = ansi.sub('', text).replace('\r\n', '\n')
        screen = []
        for line in cleaned.split('\n'):
            line = line.rsplit('\r', 1)[-1].expandtabs(8)
            screen.extend([line[i:i+112] for i in range(0, max(1, len(line)), 112)])
        image = Image.new('RGB', (1060, 620), '#111827')
        draw = ImageDraw.Draw(image)
        for row, line in enumerate(screen[-30:]):
            draw.text((20, 14 + row * 20), line, font=font, fill='#e5e7eb')
        frames.append(image)
    frames[0].save(GIF, save_all=True, append_images=frames[1:], duration=100, loop=0, optimize=True)


def capture():
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 30, 112, 0, 0))
    env = dict(os.environ, TERM='xterm-256color', COLUMNS='112', POSE_LOCALE='en')
    start, timestamp = time.monotonic(), int(time.time())
    child = subprocess.Popen(['bash', 'examples/demo/record.sh'], cwd=ROOT, env=env,
                             stdin=slave, stdout=slave, stderr=slave)
    os.close(slave)
    decoder = codecs.getincrementaldecoder('utf-8')('replace')
    events = []
    try:
        while True:
            try:
                chunk = os.read(master, 65536)
            except OSError as error:
                if error.errno == errno.EIO: break
                raise
            if not chunk: break
            events.append([round(time.monotonic()-start, 6), 'o', decoder.decode(chunk)])
    finally:
        os.close(master)
    code, duration = child.wait(), time.monotonic()-start
    if code or duration >= 60:
        raise SystemExit(f'demo did not complete within contract: exit={code}, duration={duration:.3f}\n' + ''.join(e[2] for e in events))
    header = {'version': 2, 'width': 112, 'height': 30, 'timestamp': timestamp,
              'title': 'POSE: real checks pass, missing evidence blocks closeout',
              'command': 'bash examples/demo/record.sh', 'env': {'TERM': 'xterm-256color', 'SHELL': '/bin/bash'}}
    CAST.write_text('\n'.join(json.dumps(item) for item in [header]+events)+'\n')
    render(events, duration)
    report = {'schema_version': 1, 'source_revision': subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
              'command': header['command'], 'exit_code': code, 'seconds': round(duration,3),
              'capture': 'live PTY output; original event timestamps; GIF samples at 10fps without acceleration',
              'sha256': {path.name:hashlib.sha256(path.read_bytes()).hexdigest() for path in [CAST,GIF]}}
    REPORT.write_text(json.dumps(report,indent=2)+'\n')
    validate()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    validate() if args.check else capture()
