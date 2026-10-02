"""Read-only exact-title Workshop lookup and published-file metadata capture.

Does not log in, subscribe, or assert that a title proves leaderboard identity.
"""
import concurrent.futures
import datetime
import hashlib
import json
import re
import urllib.parse
from html.parser import HTMLParser

from collect_public_sce import ROOT, get, write_json


class WorkshopItems(HTMLParser):
    def __init__(self):
        super().__init__()
        self.current_id = None
        self.items = []

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == 'a':
            match = re.search(r'/sharedfiles/filedetails/\?id=(\d+)', attrs.get('href', ''))
            self.current_id = match[1] if match else None
        if tag == 'img' and self.current_id and attrs.get('alt'):
            self.items.append({'publishedfileid': self.current_id, 'title': attrs['alt']})

    def handle_endtag(self, tag):
        if tag == 'a':
            self.current_id = None


def capture(row):
    name = row['scenario_name']
    url = 'https://steamcommunity.com/workshop/browse/?' + urllib.parse.urlencode({
        'appid': 824270, 'searchtext': name, 'browsesort': 'textsearch', 'section': 'readytouseitems'})
    result = {'scenario_name': name, 'leaderboard_id_candidate': row['leaderboard_id'],
              'source_url': url, 'retrieved_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'file_content_acquired': False, 'current_benchmark_file_verified': False, 'calibration_eligible': False}
    try:
        content = get(url)
        parser = WorkshopItems()
        parser.feed(content.decode('utf-8'))
        matches = {i['publishedfileid']: i for i in parser.items if i['title'] == name}
        result['response_sha256'] = hashlib.sha256(content).hexdigest()
        result['exact_title_ids'] = sorted(matches)
        result['status'] = 'unique_exact_title_candidate' if len(matches) == 1 else 'ambiguous_or_missing_exact_title'
        return result
    except Exception as error:
        return {**result, 'status': 'request_failed', 'error': str(error)}


def main():
    rows = json.loads((ROOT / 'data/first-family-targets.json').read_text())
    output = ROOT / 'data/workshop-candidates.json'
    previous = json.loads(output.read_text()) if output.exists() else []
    captured = {r['scenario_name']: r for r in previous if r['status'] == 'unique_exact_title_candidate'}
    pending = [r for r in rows if r['scenario_name'] not in captured]
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        for result in pool.map(capture, pending):
            captured[result['scenario_name']] = result
            write_json(output, sorted(captured.values(), key=lambda r: r['scenario_name']))
    ids = sorted({i for r in captured.values() for i in r.get('exact_title_ids', [])})
    if ids:
        import urllib.request
        url = 'https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/'
        data = {'itemcount': len(ids), **{f'publishedfileids[{i}]': id_ for i, id_ in enumerate(ids)}}
        req = urllib.request.Request(url, data=urllib.parse.urlencode(data).encode())
        try:
            with urllib.request.urlopen(req, timeout=30) as response:
                content = response.read(5_000_001)
            if len(content) > 5_000_000:
                raise ValueError('metadata response too large')
            payload = json.loads(content)
            details = []
            for item in payload['response']['publishedfiledetails']:
                # Keep only identity, file/version facts; omit unrelated user data.
                details.append({k: item.get(k) for k in ('publishedfileid', 'result', 'consumer_app_id',
                    'title', 'time_created', 'time_updated', 'file_size', 'file_url', 'hcontent_file')})
            write_json(ROOT / 'data/workshop-file-metadata.json', {'source_url': url,
                'retrieved_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
                'response_sha256': hashlib.sha256(content).hexdigest(), 'files': details,
                'file_url_is_not_guaranteed': True, 'file_to_leaderboard_binding_verified': False})
            print('Metadata files:', len(details), 'Nonempty download URLs:', sum(bool(r['file_url']) for r in details))
        except Exception as error:
            write_json(ROOT / 'data/workshop-file-metadata.json', {'status': 'request_failed', 'error': str(error), 'source_url': url})
    print('Unique exact-title candidates:', sum(r['status'] == 'unique_exact_title_candidate' for r in captured.values()), '/', len(rows))


if __name__ == '__main__':
    main()
