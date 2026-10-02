#!/usr/bin/env python3
"""Check the producer's complete asset contract, independently of docs URLs."""
import argparse
import json
from pathlib import Path
import re
import subprocess


def expected(root, version):
    config = (root / '.goreleaser.yaml').read_text()
    workflow = (root / '.github/workflows/release.yml').read_text()
    def axis(key):
        matches = re.findall(r'^\s+' + key + r': \[([\w, ]+)\]$', config, re.M)
        if len(matches) != 1:
            raise ValueError('unsupported build axis: ' + key)
        return [v.strip() for v in matches[0].split(',')]
    if 'name_template: "pose_{{ .Version }}_{{ .Os }}_{{ .Arch }}"' not in config:
        raise ValueError('unsupported archive naming contract')
    if 'formats: [tar.gz]' not in config or 'goos: windows\n        formats: [zip]' not in config:
        raise ValueError('unsupported archive format contract')
    assets = {'checksums.txt', 'checksums.txt.sigstore.json'}
    for os in axis('goos'):
        for arch in axis('goarch'):
            archive = f'pose_{version}_{os}_{arch}.' + ('zip' if os == 'windows' else 'tar.gz')
            assets.update([archive, archive + '.sigstore.json', archive + '.cdx.json', archive + '.cdx.json.sigstore.json'])
    extras = re.findall(r'^\s+- glob: ([^\s]+)\s*$', config, re.M)
    if not extras or any(re.search(r'[*?{]', name) for name in extras):
        raise ValueError('unsupported extra-file inventory')
    assets.update(extras)
    manifests = re.findall(r'^\s+package-manifests/(?:homebrew|winget)/([^\s\\]+)', workflow, re.M)
    if not manifests:
        raise ValueError('no published package manifest inventory')
    assets.update(manifests)
    return assets


def missing(root, tag, document):
    if not re.fullmatch(r'v\d+\.\d+\.\d+', tag):
        raise ValueError('release tag required')
    return expected(root, tag[1:]) - {asset['name'] for asset in document['assets']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('tag')
    parser.add_argument('--assets-json', type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    if args.assets_json:
        document = json.loads(args.assets_json.read_text())
    else:
        document = json.loads(subprocess.check_output([
            'gh', 'release', 'view', args.tag, '--repo', 'oseiaspereira88/pose', '--json', 'assets'], text=True))
    absent = missing(root, args.tag, document)
    if absent:
        raise SystemExit('published-asset-set: missing ' + ', '.join(sorted(absent)))
    print(f'published-asset-set: PASS: {len(expected(root, args.tag[1:]))} required assets present')
