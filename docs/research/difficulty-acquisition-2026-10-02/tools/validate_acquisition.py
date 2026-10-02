"""Offline integrity and join checks on acquired data; never declare fit-ready."""
import hashlib
import json

from collect_public_sce import ROOT, PREP, write_json


def load(name):
    return json.loads((ROOT / 'data' / name).read_text())


def main():
    manifest = load('download-manifest.json')
    extractions = load('structure-extractions.json')
    urls = {e['source_url']: e for e in extractions}
    previous = {p['source_url']: p for p in json.loads((PREP / 'data/structure-probes.json').read_text())['files']}
    checked, matched_previous = 0, 0
    for row in manifest:
        if row['status'] != 'downloaded_and_hash_verified':
            continue
        content = (ROOT / row['cache_file']).read_bytes()
        assert hashlib.sha256(content).hexdigest() == row['file_sha256']
        assert hashlib.sha1(f'blob {len(content)}\0'.encode() + content).hexdigest() == row['git_blob_sha1']
        extraction = urls[row['source_url']]
        assert extraction['file_sha256'] == row['file_sha256']
        assert not extraction['calibration_eligible'] and not extraction['full_mechanics_verified']
        assert extraction['geometry_features']['angular_size'] is None
        if row['source_url'] in previous:
            assert previous[row['source_url']]['file_sha256'] == row['file_sha256']
            matched_previous += 1
        checked += 1
    assert matched_previous == len(previous), 'All prior format probes must be reproduced'
    assert len(urls) == len(extractions) == checked
    targets = load('first-family-targets.json')
    assert len(targets) == len({r['leaderboard_id'] for r in targets}) == 30
    workshop = load('workshop-candidates.json')
    target_names = {r['scenario_name'] for r in targets}
    assert {r['scenario_name'] for r in workshop} == target_names
    metadata = load('workshop-file-metadata.json')
    files = {r['publishedfileid']: r for r in metadata['files']}
    for candidate in workshop:
        assert candidate['status'] == 'unique_exact_title_candidate'
        item = files[candidate['exact_title_ids'][0]]
        assert item['title'] == candidate['scenario_name']
        assert item['consumer_app_id'] == 824270 and item['result'] == 1
    vdim = load('vdim-official-playlist-catalog.json')['playlists']
    assert len(vdim) == len({r['sharecode'] for r in vdim}) == 36
    assert all(not r['scenario_difficulty_inheritance_allowed'] for r in vdim)
    secondary = load('vdim-secondary-scenario-leads.json')['rows']
    assert all(not r['calibration_eligible'] and not r['official_playlist_membership_verified'] for r in secondary)
    result = {'status': 'passed', 'downloaded_source_files_verified': checked,
              'prior_format_probes_reproduced': matched_previous,
              'first_family_target_files': len(targets), 'workshop_title_and_app_matches': len(workshop),
              'official_vdim_playlists': len(vdim), 'vdim_secondary_rows': len(secondary),
              'nonempty_workshop_file_urls': sum(bool(r['file_url']) for r in files.values()),
              'fit_eligible_samples': 0, 'difficulty_weights_fitted': False,
              'physical_geometry_validation_performed': False,
              'application_training_logic_modified': False,
              'powershell_helper_tested_on_windows': False}
    write_json(ROOT / 'data/validation-results.json', result)
    print(json.dumps(result))


if __name__ == '__main__':
    main()
