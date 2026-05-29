#!/usr/bin/env python3
"""One-shot family-prefix renamer. Walks catalog/parkervcp/*.json, looks at
upstream.path's top-level dir, and prepends a curated family prefix to the
display name when both apply:

  - The dir maps to a non-empty family in FAMILIES below, AND
  - The current name does not already start with that family.

LinuxGSM (lgsm/*) and pelican-eggs (per-game top dirs) are skipped — those
sources don't have a meaningful family-name structure to mine. IDs are
never changed, only display names. Safe to re-run; idempotent.
"""

import json
import re
from pathlib import Path

FAMILIES = {
    'minecraft': 'Minecraft',
    'terraria':  'Terraria',
    'gta':       'GTA',
    'factorio':  'Factorio',
    'rimworld':  'RimWorld',
    'doom':      'Doom',
}

CATALOG = Path(__file__).resolve().parent / 'parkervcp'


def top_dir(upstream_path: str) -> str:
    if upstream_path.startswith('game_eggs/'):
        upstream_path = upstream_path[len('game_eggs/'):]
    return upstream_path.split('/')[0] if upstream_path else ''


def main():
    renamed = 0
    for p in sorted(CATALOG.glob('*.json')):
        d = json.loads(p.read_text())
        path = d.get('upstream', {}).get('path', '')
        family = FAMILIES.get(top_dir(path), '')
        if not family:
            continue
        name = d.get('name', '').strip()
        if not name:
            continue
        if re.match(rf'^{re.escape(family)}\b', name, re.IGNORECASE):
            continue
        new_name = f'{family} {name}'
        d['name'] = new_name
        p.write_text(json.dumps(d, indent=2) + '\n')
        renamed += 1
        print(f'  {p.name:40s} {name!r:35s} -> {new_name!r}')
    print(f'renamed {renamed} entries')


if __name__ == '__main__':
    main()
