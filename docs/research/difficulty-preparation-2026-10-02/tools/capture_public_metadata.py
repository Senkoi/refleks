"""Read-only research capture. No difficulty model, game writes, or user logins.

Run from anywhere: python capture_public_metadata.py
Existing successful captures are reused; failed entries can be retried explicitly.
Only the 81 named pilot scenarios are requested, with bounded concurrency/timeouts.
"""
import concurrent.futures
import datetime
import hashlib
import json
from pathlib import Path
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "data" / "official-metadata.json"


def capture(row):
    url = "https://kovaaks.com/webapp-backend/scenario/popular?" + urllib.parse.urlencode(
        {"scenarioNameSearch": row["scenario_name"], "page": 0, "max": 25}
    )
    result = {
        "scenario_name": row["scenario_name"],
        "leaderboard_id": row["leaderboard_id"],
        "source_url": url,
        "retrieved_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    }
    try:
        with urllib.request.urlopen(url, timeout=20) as response:
            content = response.read(2_000_001)
        if len(content) > 2_000_000:
            raise ValueError("response too large")
        payload = json.loads(content)
        matches = [item for item in payload.get("data", [])
                   if item.get("leaderboardId") == row["leaderboard_id"]
                   and item.get("scenarioName") == row["scenario_name"]]
        result["response_sha256"] = hashlib.sha256(content).hexdigest()
        if len(matches) != 1:
            result["status"] = "exact_id_and_name_not_found"
            return result
        item = matches[0]
        scenario = item.get("scenario", {})
        description = scenario.get("description") or ""
        # Keep factual metadata and a fingerprint, not full authored descriptions.
        result.update({
            "status": "exact_id_and_name_matched",
            "aim_type_raw": scenario.get("aimType"),
            "authors_raw": scenario.get("authors"),
            "reported_entries": item.get("counts", {}).get("entries"),
            "reported_plays": item.get("counts", {}).get("plays"),
            "description_sha256": hashlib.sha256(description.encode()).hexdigest(),
            "file_content_acquired": False,
            "physical_file_verified": False,
        })
    except Exception as error:
        result.update({"status": "request_failed", "error": str(error)})
    return result


def main():
    candidates = json.loads((ROOT / "data" / "scenario-candidates.json").read_text())
    wanted = set((ROOT / "collection-pilot-names.txt").read_text().splitlines())
    rows = [row for row in candidates if row["scenario_name"] in wanted]
    assert len(rows) == len(wanted)
    previous = json.loads(OUTPUT.read_text()) if OUTPUT.exists() else []
    results = {row["leaderboard_id"]: row for row in previous
               if row.get("status") == "exact_id_and_name_matched"}
    pending = [row for row in rows if row["leaderboard_id"] not in results]
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        for index, result in enumerate(pool.map(capture, pending), 1):
            results[result["leaderboard_id"]] = result
            # Checkpoint so an interrupted run does not discard successful captures.
            OUTPUT.write_text(json.dumps(sorted(results.values(), key=lambda r: r["leaderboard_id"]),
                                        ensure_ascii=False, indent=2) + "\n")
            if index % 15 == 0 or result["status"] != "exact_id_and_name_matched":
                print(index, "/", len(pending), result["status"], flush=True)
    print("Captured", len(results), "Matched", sum(r["status"] == "exact_id_and_name_matched" for r in results.values()))


if __name__ == "__main__":
    main()
