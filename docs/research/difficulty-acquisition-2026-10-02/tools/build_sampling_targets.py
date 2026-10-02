"""Benchmark-first static/flick-tech sampling; raw native labels remain separate."""
import json

from collect_public_sce import ROOT, PREP, write_json


def main():
    snapshot = json.loads((PREP / "data/benchmark-snapshot.json").read_text())
    candidates = {r["leaderboard_id"]: r for r in json.loads((PREP / "data/scenario-candidates.json").read_text())}
    # Viscose Flick Tech includes both clicking and switching. These 12 exact
    # candidates were separately reviewed in the preparation family table.
    viscose_clicking = {
        'ww5t Voltaic Slightly Larger', '1w6ts reload v2 20% bigger', '1w2ts reload smallflicks larger',
        'ww4t Voltaic', '1w6ts reload v2', '1w2ts reload smallflicks slightly larger',
        'ww3t Voltaic', '1w5ts reload', '1w2ts reload smallflicks',
        'ww3t Voltaic 10% Smaller', '1w4ts Voltaic', '1w2ts reload smallflicks 25% smaller',
    }
    selected = {}
    for group in snapshot['benchmark_groups']:
        for row in group['scenarios']:
            native = row['native_category_path']
            if native != ['Clicking', 'Static'] and row['scenario_name'] not in viscose_clicking:
                continue
            target = selected.setdefault(row['leaderboard_id'], {
                **candidates[row['leaderboard_id']], 'sampling_view': 'static_clicking_and_flick_tech_candidates',
                'mechanism_category_verified': False, 'native_labels': [], 'calibration_eligible': False})
            target['native_labels'].append({
                'benchmark_id': row['benchmark_id'], 'benchmark_name': row['benchmark_name'],
                'tier': row['tier'], 'native_category_path': native, 'rank_cutoffs': row['rank_cutoffs'],
                'threshold_snapshot_response_sha256': group['response_sha256'],
                'cross_benchmark_rank_equivalence_verified': False})
    targets = list(selected.values())
    write_json(ROOT / 'data/first-family-targets.json', targets)
    (ROOT / 'first-family-names.txt').write_text('\n'.join(t['scenario_name'] for t in targets) + '\n')
    print('First-family target files:', len(targets), 'Native memberships:', sum(len(t['native_labels']) for t in targets))


if __name__ == '__main__':
    main()
