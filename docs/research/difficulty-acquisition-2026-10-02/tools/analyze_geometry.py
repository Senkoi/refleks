"""Reproducible geometry/trigger audit and conservative matched-family pairs."""
import argparse
import collections
import itertools
import json
import pathlib
import zipfile

from sce_extract import raw, split_sections
from sce_geometry import geometry_report


def profile_fields(content):
    sections, _ = split_sections(content)
    fields = {}
    occurrences = collections.Counter()
    for section in sections:
        kind = section['section']
        name = raw(section,'Name') if kind != 'Root' else 'ROOT'
        index = occurrences[(kind,name)]
        occurrences[(kind,name)] += 1
        for field in section['fields']:
            key = json.dumps([kind,name,index,field['key']])
            fields.setdefault(key, []).append(field['raw_value'])
    return fields


def pair_audit(a, b, reports, fields):
    changes = []
    for path in sorted(fields[a].keys() | fields[b].keys()):
        before, after = fields[a].get(path), fields[b].get(path)
        if before != after:
            changes.append({'path':json.loads(path),'before':before,'after':after})
    # Editorial names/descriptions and author-assigned category tags are not
    # physical mechanics. They remain explicit exclusions, never fit labels.
    # Missing newer fields are not assigned historical defaults.
    label_fields = ['Name','Description','DifficultyTag','AimSubTypeTag']
    excluded = [c for c in changes if c['path'][0]=='Root' and c['path'][-1] in label_fields]
    relevant = [c for c in changes if c not in excluded]
    size_fields = {'MainBBRadius','MainBBHeight','ProjBBRadius','ProjBBHeight'}
    size_changes = [c for c in relevant if c['path'][0]=='Character Profile' and c['path'][-1] in size_fields]
    other = [c for c in relevant if c not in size_changes]
    ma, mb = reports[a]['map'], reports[b]['map']
    same_map = bool(ma.get('semanticSHA256')) and ma.get('semanticSHA256')==mb.get('semanticSHA256')
    controlled = same_map and bool(size_changes) and not other
    directions = []
    for c in size_changes:
        if c['path'][-1] not in ['MainBBRadius','MainBBHeight']:
            continue
        try:
            va, vb = float(c['before'][0]), float(c['after'][0])
            if va > 0 and vb > 0:
                directions.append('a_smaller' if va < vb else 'b_smaller')
        except (ValueError,TypeError,IndexError):
            directions.append('unknown')
    precision = directions[0] if controlled and directions and len(set(directions))==1 else 'unknown'
    return {'a':a,'b':b,'identicalMapPayload':same_map,'sizeChanges':size_changes,
            'excludedEditorialChanges':excluded,
            'otherChangedFieldCount':len(other),'otherChangedFieldExamples':other[:8],
            'sizeOnlyConfigurationPair':controlled,
            'mainHitboxPrecisionDemandUnderFixedRuntime':precision,
            'empiricalDifficultyOrderVerified':False,
            'interpretation':'size-only config candidate; runtime still unverified' if controlled else 'confounded; no size-only difficulty conclusion'}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('archive', type=pathlib.Path)
    ap.add_argument('--targets', type=pathlib.Path, required=True)
    ap.add_argument('--output', type=pathlib.Path, required=True)
    args = ap.parse_args()
    reports, fields = {}, {}
    with zipfile.ZipFile(args.archive) as z:
        for member in sorted(z.namelist()):
            if not member.lower().endswith('.sce'):
                continue
            content = z.read(member)
            report = geometry_report(content)
            if report['name'] in reports:
                raise ValueError('duplicate internal name')
            reports[report['name']] = report
            fields[report['name']] = profile_fields(content)
    families = collections.defaultdict(list)
    for target in json.loads(args.targets.read_text()):
        if target['scenario_name'] in reports:
            families[target['family_candidate']].append(target['scenario_name'])
    pairs = [pair_audit(a,b,reports,fields) for names in families.values() for a,b in itertools.combinations(names,2)]
    result = {'scenarioCount':len(reports),'mapFormats':dict(collections.Counter(r['map']['format'] for r in reports.values())),
              'scenariosWithConditionalCenterGeometry':sum(bool(r['geometryCandidates']) for r in reports.values()),
              'sizeOnlyConfigurationPairs':sum(p['sizeOnlyConfigurationPair'] for p in pairs),
              'calibratedDifficultyModels':0,'scenarios':list(reports.values()),'familyPairs':pairs}
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(result,ensure_ascii=False,separators=(',',':'))+'\n')
    print(json.dumps({k:v for k,v in result.items() if k not in ['scenarios','familyPairs']}))


if __name__=='__main__':
    main()
