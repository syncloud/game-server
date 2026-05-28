#!/usr/bin/env python3
"""One-shot stub generator. Walks upstream egg trees at pinned SHAs and
emits catalog/<source>/<id>.json disabled-tier stubs so the catalog UI
shows full breadth. Never run by CI. Run manually when bumping pinned
upstream SHAs:

    python3 catalog/_import-stubs.py \
        --parkervcp ~/tmp/parkervcp/game_eggs \
        --parkervcp-sha fcfd5a3549769ade15127a7577d6d3c397e83b05 \
        --pelican    ~/tmp/pelican \
        --pelican-sha 34331fce33c83df752d94e2a90b1e43ca6280f82

Existing files are never overwritten — promote/edit them by hand.
"""

import argparse, csv, json, os, re, sys
from pathlib import Path

URL_LITERAL_RE = re.compile(r'https?://[^\s"\'\\)]+')
ARCHIVE_RE     = re.compile(r'\.(tar\.gz|tgz|tar\.xz|tar\.bz2|tar|zip)(?:[?#]|$)')
APPID_RE       = re.compile(r'\+app_update\s+(\d+)')
APPID_VAR_RE   = re.compile(r'\+app_update\s+\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?')
LOGIN_RE       = re.compile(r'\+login\s+(\S+)')
BASH_VAR_RE    = re.compile(r'\$\{[A-Za-z_][A-Za-z0-9_]*\}|\$[A-Z_][A-Z0-9_]+')
PORT_RE        = re.compile(r'\b([0-9]{4,5})\b')
SLUG_RE        = re.compile(r'[^a-z0-9]+')

# Reuse tokens from the existing converter logic.
NEEDS_TOOLS = ('apt install', 'apt-get install', 'apk add', 'yum install', 'dnf install')
DYNAMIC     = ('jq ', 'zgrep', 'xmllint', 'curl -sL', 'python -c', 'python3 -c')
BUILDS      = ('BuildTools.jar', 'gradlew', 'mvn ', 'cmake ', 'configure;')

def slugify(s: str) -> str:
    s = SLUG_RE.sub('-', s.lower()).strip('-')
    return s

def find_eggs(root: Path):
    for path in root.rglob('egg-*.json'):
        if path.is_file():
            yield path

def parse_egg(path: Path):
    try:
        return json.loads(path.read_text())
    except Exception as e:
        print(f'skip {path}: {e}', file=sys.stderr)
        return None

def detect_port(egg) -> int:
    for v in egg.get('variables') or []:
        env = (v.get('env_variable') or '').upper()
        if env in ('SERVER_PORT', 'PORT') or env.endswith('_PORT'):
            try:
                n = int((v.get('default_value') or '').strip())
                if n > 0:
                    return n
            except (ValueError, AttributeError):
                pass
    startup = egg.get('startup') or ''
    m = PORT_RE.search(startup)
    if m:
        n = int(m.group(1))
        if 1024 <= n <= 65535:
            return n
    return 0

def detect_protocols(egg) -> list[str]:
    s = (egg.get('startup') or '').lower()
    if 'udp' in s: return ['udp']
    if 'tcp' in s: return ['tcp']
    return ['udp']

def egg_var_value(egg, name: str) -> str | None:
    for v in egg.get('variables') or []:
        if (v.get('env_variable') or '').strip() == name:
            return (v.get('default_value') or '').strip()
    return None

def detect_steam_appid(egg, script: str) -> int | None:
    m = APPID_RE.search(script)
    if m:
        return int(m.group(1))
    m = APPID_VAR_RE.search(script)
    if m:
        val = egg_var_value(egg, m.group(1)) or ''
        if val.isdigit():
            return int(val)
    return None

def detect_login_clause(script: str) -> str:
    m = LOGIN_RE.search(script)
    return (m.group(1).strip() if m else '').lower()

def classify(egg) -> dict:
    """Return either {'tier':'experimental', 'installRecipe':..., 'requiresAccount': bool}
    or {'tier':'disabled', 'disabledReason': str}. Picked on first match — order matters."""
    script = (egg.get('scripts', {}).get('installation', {}) or {}).get('script') or ''
    entry  = (egg.get('scripts', {}).get('installation', {}) or {}).get('entrypoint', '').lower()

    if not script.strip():
        return {'tier': 'disabled', 'disabledReason': 'no install script in egg'}

    if entry and entry not in ('bash', 'sh', 'ash', '/bin/bash', '/bin/sh', '/bin/ash'):
        return {'tier': 'disabled', 'disabledReason':
                f'install entrypoint is {entry!r}, not a POSIX shell our installer can ignore'}

    appid = detect_steam_appid(egg, script)
    if appid is not None:
        login = detect_login_clause(script)
        stub = {
            'tier': 'experimental',
            'installRecipe': {'method': 'steam', 'steamAppId': appid},
        }
        if login and login != 'anonymous':
            stub['requiresAccount'] = True
        return stub

    urls = [u for u in URL_LITERAL_RE.findall(script) if ARCHIVE_RE.search(u)]
    bash_var_urls = [u for u in urls if BASH_VAR_RE.search(u)]

    if any(tok in script for tok in BUILDS):
        return {'tier': 'disabled', 'disabledReason':
                'egg builds from source (BuildTools/gradle/maven/cmake); needs a pre-built binary URL'}
    if any(tok in script for tok in DYNAMIC):
        return {'tier': 'disabled', 'disabledReason':
                'egg resolves install URL dynamically (jq/zgrep/curl pipe); needs a hand-curated static URL pin'}
    if any(tok in script for tok in NEEDS_TOOLS):
        return {'tier': 'disabled', 'disabledReason':
                'egg installs build tools via apt/apk/yum; not supported in our snap install context'}
    if bash_var_urls:
        ex = bash_var_urls[0]
        var = (BASH_VAR_RE.search(ex) or [''])[0]
        return {'tier': 'disabled', 'disabledReason':
                f'egg install URL contains unsubstituted shell variable {var}; needs a static URL pin'}
    if urls:
        return {'tier': 'disabled', 'disabledReason':
                f'candidate downloadExtract: static URL {urls[0]}; promote after testing on device'}
    return {'tier': 'disabled', 'disabledReason': 'no extractable install URL; manual recipe required'}

def make_stub(egg, source: str, commit: str, rel_path: str) -> dict:
    name = (egg.get('name') or '').strip()
    if not name:
        return None
    cls = classify(egg)
    stub = {
        'id': slugify(name),
        'name': name,
        'summary': (egg.get('description') or '').strip()[:200],
        'tier': cls['tier'],
        'defaultPort': detect_port(egg),
        'protocols': detect_protocols(egg),
        'upstream': {'commit': commit, 'path': rel_path},
    }
    if cls['tier'] == 'disabled':
        stub['disabledReason'] = cls['disabledReason']
    else:
        stub['installRecipe'] = cls['installRecipe']
        if cls.get('requiresAccount'):
            stub['requiresAccount'] = True
    return stub

CFG_KV_RE = re.compile(r'^\s*(appid|gamename|port|queryport|maxplayers)\s*=\s*"?([^"\n#]+?)"?\s*(?:#.*)?$')

def lgsm_read_cfg(path: Path) -> dict:
    out = {}
    if not path.exists(): return out
    for line in path.read_text(errors='replace').splitlines():
        m = CFG_KV_RE.match(line)
        if m: out[m.group(1)] = m.group(2).strip()
    return out

def lgsm_stubs(root: Path, sha: str):
    csv_path = root / 'lgsm/data/serverlist.csv'
    if not csv_path.exists(): return
    with csv_path.open() as f:
        for row in csv.DictReader(f):
            shortname = row['shortname'].strip()
            gameservername = row['gameservername'].strip()
            name = row['gamename'].strip()
            cfg = lgsm_read_cfg(root / 'lgsm/config-default/config-lgsm' / gameservername / '_default.cfg')
            appid = cfg.get('appid', '').strip()
            port = 0
            try:
                if cfg.get('port'):
                    n = int(cfg['port']); port = n if 0 < n < 65536 else 0
            except ValueError: pass

            if appid and appid.isdigit():
                reason = f'steam: appid {appid} (from LinuxGSM {gameservername}/_default.cfg) not yet validated for anonymous-loginnable install'
            else:
                reason = f'LinuxGSM-only entry without a Steam appid in {gameservername}/_default.cfg; needs manual install recipe'

            yield {
                'id': slugify(name),
                'name': name,
                'summary': '',
                'tier': 'disabled',
                'disabledReason': reason,
                'defaultPort': port,
                'protocols': ['udp'],
                'upstream': {'commit': sha, 'path': f'lgsm/config-default/config-lgsm/{gameservername}/_default.cfg'},
            }

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--parkervcp', type=Path)
    ap.add_argument('--parkervcp-sha', default='')
    ap.add_argument('--pelican',    type=Path)
    ap.add_argument('--pelican-sha', default='')
    ap.add_argument('--linuxgsm',   type=Path)
    ap.add_argument('--linuxgsm-sha', default='')
    ap.add_argument('--rewrite-disabled', action='store_true',
                    help='overwrite existing files whose tier is currently "disabled"')
    ap.add_argument('--out', type=Path,
                    default=Path(__file__).resolve().parent)
    args = ap.parse_args()

    sources = []
    if args.parkervcp: sources.append(('parkervcp', args.parkervcp, args.parkervcp_sha))
    if args.pelican:   sources.append(('pelican-eggs', args.pelican, args.pelican_sha))

    # Map existing id -> (path, tier). Lets --rewrite-disabled selectively
    # overwrite catalog entries that are still in their auto-generated form.
    existing = {}
    for p in args.out.glob('*/*.json'):
        try:
            t = json.loads(p.read_text()).get('tier', '')
            existing[p.stem] = (p, t)
        except Exception:
            existing[p.stem] = (p, '')

    def should_write(stub, target_dir: Path) -> bool:
        prev = existing.get(stub['id'])
        if prev is None:
            return True
        prev_path, prev_tier = prev
        if prev_path.parent != target_dir:
            # already lives in a different source dir; never cross-write a dup
            return False
        if args.rewrite_disabled and prev_tier == 'disabled':
            return True
        return False

    written = 0
    skipped_existing = 0
    skipped_unparseable = 0
    promoted_to_experimental = 0

    def write_stub(out_dir: Path, stub: dict):
        nonlocal written, promoted_to_experimental
        target = out_dir / f'{stub["id"]}.json'
        prev = existing.get(stub['id'])
        if prev and prev[1] == 'disabled' and stub['tier'] == 'experimental':
            promoted_to_experimental += 1
        target.write_text(json.dumps(stub, indent=2) + '\n')
        existing[stub['id']] = (target, stub['tier'])
        written += 1

    for source, root, sha in sources:
        out_dir = args.out / source
        out_dir.mkdir(parents=True, exist_ok=True)
        for egg_path in find_eggs(root):
            egg = parse_egg(egg_path)
            if egg is None:
                skipped_unparseable += 1; continue
            stub = make_stub(egg, source, sha, str(egg_path.relative_to(root)))
            if stub is None:
                skipped_unparseable += 1; continue
            if not should_write(stub, out_dir):
                skipped_existing += 1; continue
            write_stub(out_dir, stub)

    if args.linuxgsm:
        out_dir = args.out / 'linuxgsm'
        out_dir.mkdir(parents=True, exist_ok=True)
        for stub in lgsm_stubs(args.linuxgsm, args.linuxgsm_sha):
            if not stub['id']:
                skipped_unparseable += 1; continue
            if not should_write(stub, out_dir):
                skipped_existing += 1; continue
            write_stub(out_dir, stub)

    print(f'wrote {written} (of which {promoted_to_experimental} disabled->experimental), '
          f'skipped existing {skipped_existing}, '
          f'skipped unparseable {skipped_unparseable}')

if __name__ == '__main__':
    main()
