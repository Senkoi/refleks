"""Conservative .sce structure extraction, not a difficulty estimator.

No defaults, no inference from names, no conversion of raw world units to angles.
Repeated sections and keys retain source locations. Map payload is opaque.
"""
import collections
import hashlib
import math
import re

ROOT_FIELDS = "Name GameVersion AddedBots PlayerProfile MapName MapScale Timelimit Timescale InvincibleBots ScoreToWin ScorePerKill ScorePerDamage ScorePerTime ScoreMultAccuracy LockFOVRange LockedFOVMin LockedFOVMax LockedFOVScale".split()
CHAR_FIELDS = "Name MainBBType MainBBRadius MainBBHeight MainBBHasHead MainBBHeadRadius ProjBBType ProjBBRadius ProjBBHeight MaxHealth MaxSpeed Acceleration Gravity HealthRegenPerSec HealthRegenDelay MinRespawnDelay MaxRespawnDelay WeaponProfileNames AbilityProfileNames DisableScoring DamageKnockbackFactor".split()
BOT_FIELDS = "Name CharacterProfile DodgeProfileNames DodgeProfileWeights NoDodging UseWeapons AimingProfileNames".split()
WEAPON_FIELDS = "Name Type Category DamagePerShot TimeBetweenShots MagazineMax ReloadTime HitscanRadius".split()
BOOL_FIELDS = set("InvincibleBots ScoreMultAccuracy LockFOVRange MainBBHasHead DisableScoring NoDodging UseWeapons".split())
TEXT_FIELDS = set("Name GameVersion AddedBots PlayerProfile MapName LockedFOVScale MainBBType ProjBBType WeaponProfileNames AbilityProfileNames CharacterProfile DodgeProfileNames DodgeProfileWeights AimingProfileNames Type Category".split())


def split_sections(content):
    sections = [{"section": "Root", "line": 1, "fields": []}]
    current = sections[0]
    map_lines = []
    for line_no, line in enumerate(content.decode("utf-8-sig").splitlines(), 1):
        stripped = line.strip()
        # Map is the final opaque payload in supported files. Do not parse its
        # lines as profile headers or key/value properties.
        if current["section"] == "Map Data":
            map_lines.append(line)
            continue
        match = re.fullmatch(r"\[([^\]]+)\]", stripped)
        if match:
            current = {"section": match[1], "line": line_no, "fields": []}
            sections.append(current)
        elif "=" in line:
            key, raw = line.split("=", 1)
            current["fields"].append({"key": key.strip(), "raw_value": raw.strip(), "line": line_no})
    return sections, "\n".join(map_lines)


def values(section, key):
    return [f for f in section["fields"] if f["key"] == key]


def raw(section, key):
    vs = values(section, key)
    return vs[0]["raw_value"] if len(vs) == 1 else None


def fact(section, key):
    vs = values(section, key)
    base = {"key": key, "source_section": section["section"],
            "profile_name": raw(section, "Name"), "unit": "raw_config"}
    if not vs:
        return {**base, "status": "missing", "parsed_value": None, "raw_value": None, "line": None}
    if len(vs) != 1:
        return {**base, "status": "ambiguous_duplicate_key", "occurrences": vs, "parsed_value": None}
    v = vs[0]
    status, parsed = "explicit", v["raw_value"]
    if key in BOOL_FIELDS:
        if parsed.lower() in ("true", "false"):
            parsed = parsed.lower() == "true"
        else:
            status, parsed = "invalid", None
    elif key not in TEXT_FIELDS:
        try:
            parsed = float(parsed)
            if not math.isfinite(parsed):
                raise ValueError("non-finite")
        except ValueError:
            status, parsed = "invalid", None
    return {**base, **v, "status": status, "parsed_value": parsed}


def extract(content):
    sections, map_data = split_sections(content)
    root = sections[0]
    index = collections.defaultdict(list)
    for section in sections[1:]:
        index[(section["section"], raw(section, "Name"))].append(section)
    diagnostics, active, visited = [], {}, set()

    def resolve(kind, name, origin, stack=()):
        key = (kind, name)
        if key in stack:
            diagnostics.append({"status": "cyclic_reference", "kind": kind, "name": name, "origin": origin})
            return
        matches = index.get(key, [])
        # Preserve evidence of a case mismatch. This resolves a candidate
        # dependency, not proof of the engine's runtime name semantics.
        if not matches:
            candidates = [(k, v) for k, v in index.items()
                          if k[0] == kind and k[1] is not None and k[1].casefold() == name.casefold()]
            if len(candidates) == 1:
                key, matches = candidates[0]
                diagnostics.append({"status": "case_variant_reference", "kind": kind,
                                    "name": name, "resolved_name": key[1], "origin": origin})
                name = key[1]
        if key in stack:
            diagnostics.append({"status": "cyclic_reference", "kind": kind, "name": name, "origin": origin})
            return
        if len(matches) != 1:
            diagnostics.append({"status": "unresolved_reference" if not matches else "ambiguous_profile_name",
                                "kind": kind, "name": name, "origin": origin})
            return
        if key in visited:
            return
        visited.add(key)
        section = matches[0]
        fields = {"Bot Profile": BOT_FIELDS, "Character Profile": CHAR_FIELDS,
                  "Weapon Profile": WEAPON_FIELDS}.get(kind)
        active[key] = {"section": kind, "profile_name": name, "line": section["line"],
                       "fields": [fact(section, f) for f in fields] if fields else section["fields"]}
        refs = {"Bot Profile": [("CharacterProfile", "Character Profile"), ("DodgeProfileNames", "Dodge Profile")],
                "Character Profile": [("WeaponProfileNames", "Weapon Profile"), ("AbilityProfileNames", "Ability Profile")],
                "Weapon Ability Profile": [("WeaponProfile", "Weapon Profile")],
                "Bot Rotation Profile": [("ProfileNames", "Bot Profile")]}.get(kind, [])
        for field, dest in refs:
            value = raw(section, field)
            if value is None:
                if field in ("CharacterProfile", "ProfileNames"):
                    diagnostics.append({"status": "missing_or_ambiguous_reference_field", "profile": name, "field": field})
                continue
            for slot, token in enumerate(value.split(";")):
                token = token.strip()
                if not token:
                    continue
                if dest == "Ability Profile":
                    ability_kind = {"abilmov": "Movement Ability Profile", "abilwep": "Weapon Ability Profile", "abilmelee": "Melee Ability Profile"}.get(token.rsplit(".", 1)[-1])
                    if ability_kind:
                        resolve(ability_kind, token.rsplit(".", 1)[0], {"profile": name, "field": field, "slot": slot}, stack + (key,))
                    else:
                        diagnostics.append({"status": "unsupported_ability_reference", "token": token, "profile": name})
                    continue
                suffix = ".bot" if dest == "Bot Profile" else ".wpn" if dest == "Weapon Profile" else None
                target_name = token[:-len(suffix)] if suffix and token.endswith(suffix) else token
                if dest == "Bot Profile" and token.endswith(".rot"):
                    resolve("Bot Rotation Profile", token[:-4], {"profile": name, "field": field, "slot": slot}, stack + (key,))
                else:
                    resolve(dest, target_name, {"profile": name, "field": field, "slot": slot}, stack + (key,))

    player = raw(root, "PlayerProfile")
    if player:
        resolve("Character Profile", player, {"section": "Root", "field": "PlayerProfile"})
    else:
        diagnostics.append({"status": "missing_or_ambiguous_player_profile"})
    added = raw(root, "AddedBots")
    slots = added.split(";") if added is not None else None
    if slots is None:
        diagnostics.append({"status": "missing_or_ambiguous_added_bots"})
    else:
        for slot, token in enumerate(slots):
            token = token.strip()
            if not token:
                continue
            if token.endswith(".bot"):
                resolve("Bot Profile", token[:-4], {"field": "AddedBots", "slot": slot})
            elif token.endswith(".rot"):
                resolve("Bot Rotation Profile", token[:-4], {"field": "AddedBots", "slot": slot})
            else:
                diagnostics.append({"status": "unsupported_bot_slot", "token": token, "slot": slot})
    profile_counts = collections.Counter(s["section"] for s in sections[1:] if s["section"] != "Map Data")
    return {"file_sha256": hashlib.sha256(content).hexdigest(), "internal_name": raw(root, "Name"),
            "root_fields": [fact(root, f) for f in ROOT_FIELDS],
            "added_bot_slots": slots, "initial_bot_slot_count": sum(bool(s.strip()) for s in slots) if slots is not None else None,
            "all_profile_counts": dict(profile_counts), "reachable_profiles": list(active.values()),
            "reference_diagnostics": diagnostics,
            "map_payload": {"present": bool(map_data), "payload_text_sha256": hashlib.sha256(map_data.encode()).hexdigest() if map_data else None,
                            "line_count": len(map_data.splitlines()), "geometry_status": "unsupported"},
            "geometry_features": {"angular_size": None, "angular_speed": None, "transition_angle": None},
            "geometry_status": "unknown_pending_map_and_engine_validation",
            "reference_scope": "root_bot_rotation_character_dodge_weapon_movement_weapon_melee_abilities; weapon_reachability_is_not_active_firing",
            "full_mechanics_verified": False, "calibration_eligible": False}
