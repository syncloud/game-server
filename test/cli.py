import json
import shlex
import time


def run(device, *args, want_json=True):
    """Invoke games.cli on the device. Auto-appends --json when want_json is
    set and --json isn't already in args. Returns the parsed JSON (None for
    empty output) or the raw stdout if want_json=False."""
    parts = ['games.cli'] + list(args)
    if want_json and '--json' not in args:
        parts.append('--json')
    out = device.run_ssh(' '.join(shlex.quote(p) for p in parts))
    if not want_json:
        return out
    out = (out or '').strip()
    if not out:
        return None
    return json.loads(out)


def run_text(device, *args):
    return run(device, *args, want_json=False)


def wait_status(device, name, target, timeout):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        s = run(device, 'server', 'show', name)
        last = s.get('status') if s else None
        if last == target:
            return s
        if last == 'install-error':
            raise AssertionError('install errored: {!r}'.format(s))
        time.sleep(2)
    raise AssertionError('timeout waiting for status={} on {}, last={}'.format(target, name, last))


def wait_a2s(device, name, timeout):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        try:
            return run(device, 'server', 'query', name)
        except Exception as e:
            last = str(e)
            time.sleep(3)
    raise AssertionError('timeout waiting for A2S response, last={}'.format(last))
