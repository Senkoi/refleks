"""Download exact-name historical candidates and prior format probes.

All downloads remain in ignored .cache; public outputs contain hashes and facts.
Only immutable commit URLs are used. Same-name does not verify current version.
"""
import argparse
import concurrent.futures
import datetime
import hashlib
import json
from pathlib import Path
import urllib.parse
import urllib.request

from sce_extract import extract

ROOT = Path(__file__).resolve().parents[1]
PREP = ROOT.parent / "difficulty-preparation-2026-10-02"
SOURCES = [
    ("fvolpe83/Scenarios", "6a4563d241e61a62020f76796762df5ae8817cc8", "2020-09-23T06:26:05Z"),
    ("MBHuman/Scenarios", "1db6bfdec8cc42164ca9ff57dd9d3c82cfaf2137", "2020-11-21T16:47:14Z"),
    ("0xh34z/FPSAimTrainer", "5f8cd5b54dcadf3c1e0a2138bb34fb7d54306bfc", "2026-07-22T18:37:25Z"),
]


def write_json(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")


def get(url):
    req = urllib.request.Request(url, headers={"User-Agent": "refleks-difficulty-research"})
    with urllib.request.urlopen(req, timeout=30) as response:
        content = response.read(20_000_001)
    if len(content) > 20_000_000:
        raise ValueError("20 MB response limit")
    return content


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--tree-cache", type=Path, help="optional existing repository-tree snapshots")
    args = parser.parse_args()
    (ROOT / ".cache").mkdir(parents=True, exist_ok=True)
    candidates = json.loads((PREP / "data/scenario-candidates.json").read_text())
    names = {r["scenario_name"]: r for r in candidates}
    probes = json.loads((PREP / "data/structure-probes.json").read_text())["files"]
    probe_urls = {r["source_url"]: r for r in probes}
    plans, source_audits = {}, []
    for repo, commit, date in SOURCES:
        url = f"https://api.github.com/repos/{repo}/git/trees/{commit}?recursive=1"
        audit = {"repo": repo, "commit": commit, "commit_date": date, "tree_url": url}
        try:
            cached = args.tree_cache / (repo.replace("/", "_") + ".json") if args.tree_cache else None
            if cached and cached.exists():
                content = cached.read_bytes()
                tree = json.loads(content)
                if tree["sha"] != commit:
                    raise ValueError("tree cache does not match pinned commit")
            else:
                content = get(url)
                tree = json.loads(content)
            if tree.get("truncated"):
                raise ValueError("truncated tree cannot prove absence")
            entries = [x for x in tree["tree"] if x["type"] == "blob" and x["path"].endswith(".sce")]
            audit.update({"status": "complete_tree_checked", "tree_sha256": hashlib.sha256(content).hexdigest(),
                          "sce_count": len(entries), "exact_name_matches": 0})
            for entry in entries:
                name = Path(entry["path"]).stem
                source_url = f"https://raw.githubusercontent.com/{repo}/{commit}/" + urllib.parse.quote(entry["path"], safe="/")
                if name not in names and source_url not in probe_urls:
                    continue
                if name in names:
                    audit["exact_name_matches"] += 1
                plans[source_url] = {"source_repo": repo, "source_commit": commit, "source_commit_date": date,
                                     "source_path": entry["path"], "source_url": source_url, "git_blob_sha1": entry["sha"],
                                     "expected_byte_count": entry.get("size"), "scenario_name_candidate": name,
                                     "leaderboard_id_candidate": names.get(name, {}).get("leaderboard_id"),
                                     "purpose": "historical_exact_name_candidate" if name in names else "format_probe"}
        except Exception as error:
            audit.update({"status": "request_or_validation_failed", "error": str(error)})
        source_audits.append(audit)

    def download(plan):
        result = {**plan, "retrieved_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  "current_benchmark_file_verified": False, "calibration_eligible": False}
        cache = ROOT / ".cache" / (plan["git_blob_sha1"] + ".sce")
        try:
            content = cache.read_bytes() if cache.exists() else get(plan["source_url"])
            blob = hashlib.sha1(f"blob {len(content)}\0".encode() + content).hexdigest()
            if blob != plan["git_blob_sha1"]:
                raise ValueError("Git blob hash mismatch")
            if len(content) != plan["expected_byte_count"]:
                raise ValueError("byte count mismatch")
            extraction = extract(content)
            # Archive names themselves may differ from internal Name; preserve
            # that conflict rather than silently attaching a leaderboard.
            result.update({"status": "downloaded_and_hash_verified", "file_sha256": extraction["file_sha256"],
                           "byte_count": len(content), "cache_file": str(cache.relative_to(ROOT)),
                           "internal_name": extraction["internal_name"],
                           "internal_name_matches_candidate": extraction["internal_name"] == plan["scenario_name_candidate"]})
            cache.write_bytes(content)
            return result, {"source_url": plan["source_url"], **extraction}
        except Exception as error:
            return {**result, "status": "download_or_validation_failed", "error": str(error)}, None

    downloads, extractions = [], []
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        for result, extraction in pool.map(download, plans.values()):
            downloads.append(result)
            if extraction:
                extractions.append(extraction)
            write_json(ROOT / "data/download-manifest.json", downloads)
    write_json(ROOT / "data/source-coverage.json", source_audits)
    write_json(ROOT / "data/structure-extractions.json", extractions)
    historical = {r["scenario_name_candidate"] for r in downloads if r["status"] == "downloaded_and_hash_verified" and r["internal_name_matches_candidate"]}
    pilot = set((PREP / "collection-pilot-names.txt").read_text().splitlines())
    coverage = [{**r, "pilot": r["scenario_name"] in pilot,
                 "historical_exact_name_file_acquired": r["scenario_name"] in historical,
                 "current_file_status": "not_acquired", "fit_status": "blocked_current_file_and_label_evidence"} for r in candidates]
    write_json(ROOT / "data/candidate-file-coverage.json", coverage)
    (ROOT / "missing-current-pilot-names.txt").write_text("\n".join(sorted(pilot)) + "\n")
    summary = {"downloaded_source_files": len(extractions), "unique_file_sha256": len({e["file_sha256"] for e in extractions}),
               "candidate_exact_names_with_historical_file": sum(r["historical_exact_name_file_acquired"] for r in coverage),
               "pilot_exact_names_with_historical_file": sum(r["pilot"] and r["historical_exact_name_file_acquired"] for r in coverage),
               "pilot_current_files_verified": 0, "fit_eligible_samples": 0,
               "all_candidates": len(coverage), "pilot_candidates": len(pilot),
               "download_failures": sum(r["status"] != "downloaded_and_hash_verified" for r in downloads),
               "all_source_trees_checked": all(s["status"] == "complete_tree_checked" for s in source_audits),
               "source_coverage_scope": "only_the_pinned_repositories_not_all_public_sources"}
    write_json(ROOT / "data/acquisition-summary.json", summary)
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == "__main__":
    main()
