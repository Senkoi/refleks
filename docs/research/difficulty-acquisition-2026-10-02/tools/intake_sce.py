"""Build a reproducible mechanism snapshot from a user-provided ZIP.

Tags describe declared demands, not calibrated difficulty or verified runtime.
No scenario-name rules, world-to-angle conversion or playlist-tier inheritance.
"""
import argparse
import collections
import hashlib
import json
import math
import pathlib
import zipfile

from sce_extract import extract, raw, split_sections

RULES = {
    "short_transfer": ["two close, small targets"],
    "wide_transfer": ["wide curved wall", "wideflicks"],
    "micro_adjustment": ["microadjustments", "wideflicks and micros", "two close, small targets"],
    "precision": ["precision on tiny targets", "extra care", "two close, small targets"],
    "time_pressure": ["before they reach you", "before they despawn or collide", "before it despawns", "falls to the ground"],
    "reflex_window": ["short-lived target", "lifetime: 500 ms"],
    "pacing": ["pacing to increase"],
    "phased_targets": ["number of targets increases every", "after killing 3 big bots"],
    "tracking": ["hitscan tracking", "track vertically", "tracking while revolving"],
    "projectile": ["fast projectile", "midair rockets"],
}


def mechanism(content):
    parsed = extract(content)
    sections, _ = split_sections(content)
    root = sections[0]
    description = (raw(root, "Description") or "").lower()
    tags = []
    evidence = []
    for tag, phrases in RULES.items():
        matched = next((phrase for phrase in phrases if phrase in description), None)
        if matched:
            tags.append(tag)
            evidence.append({"tag": tag, "source": "author_description", "field": "Description", "matched": matched})
    # An explicit accuracy multiplier describes a scoring constraint only.
    # It is not a proof of SQRT scoring, nor a normalized difficulty value.
    if (raw(root, "ScoreMultAccuracy") or "").lower() == "true":
        tags.append("accuracy_constraint")
        evidence.append({"tag": "accuracy_constraint", "source": "explicit_field", "field": "ScoreMultAccuracy"})
    player = raw(root, "PlayerProfile")
    chars = [s for s in sections if s["section"] == "Character Profile" and raw(s, "Name") == player]
    weapons = (raw(chars[0], "WeaponProfileNames") or "").split(";") if len(chars) == 1 else []
    weapon_names = {w.removesuffix(".wpn") for w in weapons if w}
    weapon_facts = []
    for s in sections:
        if s["section"] != "Weapon Profile" or raw(s, "Name") not in weapon_names:
            continue
        fields = {k: raw(s, k) for k in ["Name", "Type", "TimeBetweenShots", "MagazineMax", "AmmoPerShot", "AmmoReloadedOnKill", "ReloadTimeFromEmpty", "ReloadTimeFromPartial"]}
        weapon_facts.append(fields)
        try:
            constrained = float(fields["MagazineMax"]) > 0 and float(fields["AmmoPerShot"]) > 0 and float(fields["AmmoReloadedOnKill"]) > 0 and float(fields["ReloadTimeFromEmpty"]) > 0
        except (ValueError, TypeError):
            constrained = False
        if constrained and "reload_constraint" not in tags:
            tags.append("reload_constraint")
            evidence.append({"tag": "reload_constraint", "source": "explicit_player_weapon_fields", "profile": raw(s, "Name")})
    try:
        limit = float(raw(root, "Timelimit"))
        seconds = math.ceil(limit) if 10 <= limit <= 3600 else 0
    except (ValueError, TypeError):
        seconds = 0
    return parsed, {
        "fileSHA256": parsed["file_sha256"], "tags": sorted(tags),
        "declaredSkill": "static" if "static clicking scenario" in description else "",
        "declaredSeconds": seconds,
        "status": "snapshot_unverified_runtime", "role": "unknown",
        "geometryStatus": "unknown", "angularSize": None, "transitionAngle": None,
        "evidence": evidence, "playerWeaponFacts": weapon_facts,
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("archive", type=pathlib.Path)
    ap.add_argument("--snapshot", type=pathlib.Path, required=True)
    ap.add_argument("--inventory", type=pathlib.Path, required=True)
    args = ap.parse_args()
    rows, inventory = [], []
    seen = set()
    with zipfile.ZipFile(args.archive) as archive:
        for member in sorted(archive.namelist()):
            if not member.lower().endswith(".sce"):
                continue
            # Never extract archive paths to disk.
            parsed, profile = mechanism(archive.read(member))
            name = parsed["internal_name"]
            if not name or name.casefold() in seen:
                raise ValueError(f"Missing or duplicate internal name: {member}")
            seen.add(name.casefold())
            rows.append({"name": name, "mechanics": profile})
            inventory.append({"member": member, "name": name, "fileSHA256": parsed["file_sha256"],
                              "tags": profile["tags"], "rootFields": parsed["root_fields"],
                              "profileCounts": parsed["all_profile_counts"],
                              "referenceDiagnostics": parsed["reference_diagnostics"],
                              "fullMechanicsVerified": False, "calibrationEligible": False})
    digest = hashlib.sha256(args.archive.read_bytes()).hexdigest()
    for path, value in [(args.snapshot, {"archiveSHA256": digest, "scenarios": rows}),
                        (args.inventory, {"archiveSHA256": digest, "scenarioCount": len(rows), "scenarios": inventory})]:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"scenarios": len(rows), "tagged": sum(bool(r['mechanics']['tags']) for r in rows),
                      "tags": dict(collections.Counter(t for r in rows for t in r['mechanics']['tags']))}))


if __name__ == "__main__":
    main()
