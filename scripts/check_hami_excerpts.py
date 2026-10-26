"""Verify lesson snippets against pinned Git objects, not working-tree files."""
import argparse
import json
from pathlib import Path
import subprocess
import textwrap

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--kubernetes', type=Path, required=True)
p.add_argument('--hami', type=Path, required=True)
p.add_argument('--kubelet', type=Path, required=True)
a = p.parse_args()
roots = {'https://github.com/kubernetes/kubernetes': a.kubernetes,
         'https://github.com/Project-HAMi/HAMi': a.hami,
         'https://github.com/kubernetes/kubelet': a.kubelet}
records = json.loads((Path(__file__).resolve().parents[1] / 'courses/hami/01-excerpts.json').read_text())
for item in records:
    raw = subprocess.check_output(['git', '-C', str(roots[item['repository']]),
                                   'show', item['revision'] + ':' + item['path']], text=True)
    code = textwrap.dedent('\n'.join(raw.splitlines()[item['start']-1:item['end']])).expandtabs(2)
    if code != item['code']:
        raise SystemExit('Source mismatch: ' + item['path'] + ':' + str(item['start']))
print('Verified', len(records), 'excerpts against pinned commits; this is not runtime validation.')
