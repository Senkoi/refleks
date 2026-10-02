"""Extract factual VDIM playlist identities, never inherit difficulty labels.

Full document and alternate-engine definitions stay outside public outputs.
The FpsAimForge bundle is a secondary, unverified playlist lead only.
"""
import argparse
import hashlib
import json
import re
from html.parser import HTMLParser
from pathlib import Path

from collect_public_sce import ROOT, PREP, get, write_json

DOC_ID = "1R4IyJqYmprRauaACt6bah7YzOOVuG6GqXlB03clNHeU"
DOC_URL = f"https://docs.google.com/document/d/{DOC_ID}/export?format=html"
SECONDARY_URL = "https://raw.githubusercontent.com/mjohns/FpsAimForge/c72cd864cfd11be4fc238078c4ee1fc6f0e0a1c9/src/resources/bundles/VDIM.bundle.json"


class Paragraphs(HTMLParser):
    def __init__(self):
        super().__init__()
        self.paragraphs, self.part = [], None
        self.skipped = 0

    def handle_starttag(self, tag, attrs):
        if tag in ("script", "style"):
            self.skipped += 1
        if tag in ("p", "h1", "h2", "h3", "h4"):
            self.part = []

    def handle_data(self, data):
        if self.part is not None and not self.skipped:
            self.part.append(data)

    def handle_endtag(self, tag):
        if tag in ("script", "style"):
            self.skipped -= 1
        if tag in ("p", "h1", "h2", "h3", "h4") and self.part is not None:
            self.paragraphs.append("".join(self.part))
            self.part = None


def capture_document(content):
    parser = Paragraphs()
    parser.feed(content.decode("utf-8"))
    playlists, pending = [], None
    day = None
    weekdays = {"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"}
    for i, paragraph in enumerate(parser.paragraphs):
        normalized = " ".join(paragraph.split())
        if normalized in weekdays:
            day = normalized.title()
        match = re.fullmatch(r"> PLAY (ENTRY|NOVICE|ADEPT|INTERMEDIATE|ADVANCED|ELITE) (CLICKING|TRACKING|SWITCHING) (I|II)", normalized)
        if match:
            pending = {"day": day, "recommended_player_tier": match[1].title(),
                       "daily_category": match[2].title(), "daily_category_part": match[3],
                       "source_paragraph_index": i, "source_url": DOC_URL,
                       "scenario_difficulty_inheritance_allowed": False}
        codes = re.findall(r"\bKovaaKs[A-Za-z0-9]+\b", normalized)
        if codes and pending:
            if len(codes) != 1:
                raise ValueError("ambiguous sharecodes")
            playlists.append({**pending, "sharecode": codes[0], "individual_scenarios_verified": False,
                              "warmup_roles_verified": False})
            pending = None
    if len(playlists) != 36 or len({p["sharecode"] for p in playlists}) != 36:
        raise ValueError("expected six days x six tiers with distinct sharecodes; source changed")
    return playlists


def secondary_rows(content):
    bundle = json.loads(content)
    candidates = json.loads((PREP / "data/scenario-candidates.json").read_text())
    names = {r["scenario_name"]: r for r in candidates}
    rows = []
    for playlist in bundle["playlists"]:
        for position, item in enumerate(playlist["def"]["items"], 1):
            # "VDIM " is this app's bundle namespace, not a KovaaK scenario prefix.
            original = item["scenario"]
            name = original.removeprefix("VDIM ")
            benchmark = names.get(name)
            rows.append({"playlist_name": playlist["name"], "position": position,
                         "scenario_name_candidate": name, "secondary_name_raw": original,
                         "repeat_count_in_secondary_copy": item["numPlays"],
                         "leaderboard_id_candidate": benchmark["leaderboard_id"] if benchmark else None,
                         "role": "benchmark_name_candidate" if benchmark else "unknown",
                         "benchmark_memberships_candidate": benchmark["memberships"] if benchmark else [],
                         "role_verified": False, "official_playlist_membership_verified": False,
                         "source_url": SECONDARY_URL,
                         "mechanical_definitions_imported": False, "calibration_eligible": False})
    return rows


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--document-cache", type=Path)
    parser.add_argument("--secondary-cache", type=Path)
    args = parser.parse_args()
    document = args.document_cache.read_bytes() if args.document_cache else get(DOC_URL)
    secondary = args.secondary_cache.read_bytes() if args.secondary_cache else get(SECONDARY_URL)
    playlists, rows = capture_document(document), secondary_rows(secondary)
    write_json(ROOT / "data/vdim-official-playlist-catalog.json", {
        "source_url": DOC_URL, "source_sha256": hashlib.sha256(document).hexdigest(),
        "author_entrypoint": "https://bit.ly/VDIMkvksS5", "playlists": playlists,
        "role_policy": {"valid_roles": ["warmup", "skill_isolation", "overload", "benchmark", "unknown"],
                        "position_or_repeat_count_is_role_evidence": False,
                        "same_playlist_tier_means_equal_scenario_difficulty": False,
                        "playlist_order_is_global_difficulty_order": False,
                        "warmup_role_can_be_used_as_supervised_difficulty_label": False}})
    write_json(ROOT / "data/vdim-secondary-scenario-leads.json", {
        "source_url": SECONDARY_URL, "source_sha256": hashlib.sha256(secondary).hexdigest(),
        "source_status": "alternate_engine_port_unverified_against_official_playlists",
        "rows": rows})
    print("Official playlists:", len(playlists), "Secondary rows:", len(rows),
          "Secondary unique scenarios:", len({r['scenario_name_candidate'] for r in rows}),
          "Benchmark-name candidates:", sum(r['role'] == 'benchmark_name_candidate' for r in rows))


if __name__ == "__main__":
    main()
