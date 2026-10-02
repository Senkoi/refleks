"""Map facts and conditional geometry; never authoritative engine geometry.

JSON spawn coordinates are explicit facts. Legacy omitted team flags retain
unknown values. Coordinate/scale/camera/collision semantics require validation.
"""
import collections
import hashlib
import json
import math
import itertools
import re

from sce_extract import extract, raw, split_sections


def vector(value):
    if not isinstance(value, str):
        return None
    tokens = value.replace(',', ' ').split()
    if len(tokens) != 3:
        return None
    try:
        result = [float(t) for t in tokens]
        return result if all(math.isfinite(t) for t in result) else None
    except ValueError:
        return None


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f'duplicate JSON key: {key}')
        result[key] = value
    return result


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, ensure_ascii=False).encode()).hexdigest()


def parse_map(payload):
    spawns, objects, diagnostics = [], collections.Counter(), []
    if payload.lstrip().startswith('{'):
        try:
            data = json.loads(payload, object_pairs_hook=unique_object,
                              parse_constant=lambda x: (_ for _ in ()).throw(ValueError(x)))
            if not isinstance(data.get('objects'), list):
                raise ValueError('missing objects array')
            for index, obj in enumerate(data['objects']):
                if not isinstance(obj, dict):
                    raise ValueError('invalid map object')
                if obj.get('type') != 'gameObject':
                    continue
                name = obj.get('name')
                objects[str(name)] += 1
                if name != 'SpawnPoint':
                    continue
                props = {}
                for prop in obj.get('properties', []):
                    if prop['name'] in props:
                        raise ValueError('duplicate spawn property')
                    props[prop['name']] = prop['value']
                position = vector(obj.get('location'))
                if position is None:
                    diagnostics.append({'status': 'invalid_spawn_position', 'object': index})
                spawns.append({'object': index, 'position': position,
                               'rotation': vector(obj.get('rotation')), 'properties': props})
            fmt, semantic_hash = 'kmc_json', digest(data)
        except (ValueError, TypeError, KeyError) as error:
            return {'format': 'invalid_json', 'status': 'invalid', 'diagnostics': [str(error)], 'spawns': []}
    elif payload.startswith('reflex map version '):
        # Entity blocks are delimited by indentation. Brush vertices and faces
        # are not entities; missing team flags are NOT filled with defaults.
        lines = payload.splitlines()
        for index, line in enumerate(lines):
            if line.strip() != 'entity':
                continue
            indent = len(line) - len(line.lstrip())
            fields = []
            for child in itertools.islice(lines, index+1, None):
                if child.strip() and len(child) - len(child.lstrip()) <= indent:
                    break
                if child.strip():
                    fields.append(child.strip())
            types = [f[5:] for f in fields if f.startswith('type ')]
            if len(types) != 1:
                diagnostics.append({'status': 'ambiguous_entity_type', 'line': index+1})
                continue
            objects[types[0]] += 1
            if types[0] != 'PlayerSpawn':
                continue
            props = {}
            for field in fields:
                tokens = field.split(' ', 2)
                if len(tokens) == 3:
                    _, key, value = tokens
                    if key in props:
                        diagnostics.append({'status': 'duplicate_entity_property', 'line': index+1, 'key': key})
                        props[key] = None
                    else:
                        props[key] = value
            spawns.append({'line': index+1, 'position': vector(props.get('position')),
                           'rotation': vector(props.get('angles')), 'properties': props})
        fmt, semantic_hash = 'reflex_text', hashlib.sha256(payload.encode()).hexdigest()
    else:
        return {'format': 'unsupported', 'status': 'unsupported', 'diagnostics': [], 'spawns': []}
    positions = [tuple(s['position']) for s in spawns if s['position'] is not None]
    return {'format': fmt, 'status': 'parsed_facts_only', 'semanticSHA256': semantic_hash,
            'objectCounts': dict(objects), 'spawnCount': len(spawns), 'uniquePositionCount': len(set(positions)),
            'diagnostics': diagnostics, 'spawns': spawns}


def eligible_json_spawns(parsed_map, team, character):
    if parsed_map['format'] != 'kmc_json':
        return None
    try:
        team = int(team)
    except (ValueError, TypeError):
        return None
    if team < 0 or team > 32:
        return None
    result = []
    for spawn in parsed_map['spawns']:
        p = spawn['properties']
        mask, permitted, weight = p.get('TeamMask'), p.get('PermittedCharacterProfiles'), p.get('Weight')
        # Team-mask encoding is a documented editor concept, but this numeric
        # bit interpretation remains an explicit hypothesis, including team 0.
        if type(mask) is not int or mask < 0 or permitted is None or spawn['position'] is None:
            return None
        if type(weight) not in (int,float) or not math.isfinite(weight) or weight < 0:
            return None
        if weight == 0:
            continue
        allowed = [s.strip() for s in re.split('[,;]', permitted) if s.strip()] if isinstance(permitted, str) else None
        if allowed is None:
            return None
        if (team == 0 or mask & (1 << (team-1))) and (not allowed or character in allowed):
            result.append(spawn)
    return result


def center_candidates(player_positions, target_positions, radius, scale):
    """Exact spherical angular diameter for the stated center/scale model."""
    if not player_positions or not target_positions or not all(math.isfinite(v) and v>0 for v in [radius,scale]):
        return None
    sizes, transitions = [], []
    for player in player_positions:
        directions = []
        for target in target_positions:
            v = [(a-b)*scale for a, b in zip(target, player)]
            distance = math.sqrt(sum(x*x for x in v))
            if distance <= radius:
                return None
            directions.append([x/distance for x in v])
            sizes.append(math.degrees(2*math.asin(radius/distance)))
        for i, a in enumerate(directions):
            for b in directions[i+1:]:
                transitions.append(math.degrees(math.acos(max(-1, min(1, sum(x*y for x,y in zip(a,b)))))))
    return {'angularDiameterDegrees': {'min': min(sizes), 'max': max(sizes)},
            'spawnCenterPairAngleDegrees': {'min': min(transitions), 'max': max(transitions)} if transitions else None}


def geometry_report(content):
    sections, payload = split_sections(content)
    root, structural = sections[0], extract(content)
    parsed_map = parse_map(payload)
    report = {'name': raw(root, 'Name'), 'fileSHA256': structural['file_sha256'],
              'map': {k:v for k,v in parsed_map.items() if k != 'spawns'},
              'rootSpawnFacts': {k:raw(root,k) for k in ['PlayerProfile', 'PlayerTeam', 'BotTeams', 'AddedBots', 'MapScale', 'Timescale']},
              'abilityConditions': [], 'characterSpawnFacts': [], 'rotationFacts': [],
              'geometryCandidates': [], 'authoritativeGeometry': None, 'calibrationEligible': False}
    active = {(p['section'], p['profile_name']) for p in structural['reachable_profiles']}
    by_profile = {}
    for section in sections[1:]:
        kind, name = section['section'], raw(section, 'Name')
        if (kind, name) not in active:
            continue
        if kind == 'Character Profile':
            by_profile[name] = section
            keys = ['MainBBType', 'MainBBRadius', 'MainBBHeight', 'MaxSpeed', 'MaxHealth', 'MinRespawnDelay', 'MaxRespawnDelay',
                    'SpawnOffsetMin', 'SpawnOffsetMax', 'SpawnXOffset', 'SpawnYOffset', 'VerticalSpawnOffset',
                    'CameraOffset', 'BlockedSpawnRadius', 'BlockSpawnFOV', 'BlockSpawnDistance', 'InvertBlockedSpawn', 'AbilityProfileNames']
            report['characterSpawnFacts'].append({'name': name, 'fields': {k:raw(section,k) for k in keys}, 'line': section['line']})
        if 'Ability Profile' in kind:
            report['abilityConditions'].append({'name': name, 'kind': kind, 'line': section['line'], 'fields': section['fields']})
        if kind == 'Bot Rotation Profile':
            report['rotationFacts'].append({'name': name, 'line': section['line'], 'fields': section['fields']})
    player = eligible_json_spawns(parsed_map, raw(root,'PlayerTeam'), raw(root,'PlayerProfile'))
    if not player:
        report['geometryBlockedBy'] = 'legacy_flags_or_player_spawn_semantics_unknown'
        return report
    teams = (raw(root,'BotTeams') or '').split(';')
    slots = structural['added_bot_slots'] or []
    if len(teams) != len(slots):
        report['geometryBlockedBy'] = 'bot_team_slot_alignment_unknown'
        return report
    bots = {p['profile_name']: p for p in structural['reachable_profiles'] if p['section']=='Bot Profile'}
    for token, team in zip(slots, teams):
        matches = [p for name,p in bots.items() if name.casefold()==token.removesuffix('.bot').casefold()]
        if len(matches) != 1:
            continue # Rotation slots require phase-specific runtime validation.
        char_names = [f['parsed_value'] for f in matches[0]['fields'] if f['key']=='CharacterProfile']
        if len(char_names) != 1:
            continue
        char = next((v for k,v in by_profile.items() if k.casefold()==str(char_names[0]).casefold()), None)
        if char is None:
            continue
        name = raw(char, 'Name')
        if any(c['character']==name and c['team']==team for c in report['geometryCandidates']):
            continue
        target = eligible_json_spawns(parsed_map, team, name)
        if not target:
            continue
        if len(target) > 256 or len(player) > 16:
            continue # Bounded all-pairs analysis; no silent sampling.
        try:
            radius, height, scale = [float(raw(section,key)) for section,key in [(char,'MainBBRadius'),(char,'MainBBHeight'),(root,'MapScale')]]
        except (ValueError, TypeError):
            continue
        if not all(math.isfinite(v) and v>0 for v in [radius,height,scale]) or raw(char,'MainBBType') != 'Spheroid' or not math.isclose(height,2*radius,rel_tol=1e-5):
            continue
        ps = [s['position'] for s in player]; ts = [s['position'] for s in target]
        # No empirical distributions: these are extrema of base spawn centers,
        # ignoring offsets, camera displacement, rejection and collision rules.
        report['geometryCandidates'].append({'character':name,'team':team,'eligiblePlayerSpawns':len(ps),'eligibleTargetSpawns':len(ts),
            'assumptions':['bitmask_team_encoding','profile_list_filter','spawn_centers_only','map_axis_order_preserved','object_scale_and_rotation_ignored','zero_camera_and_offsets','spherical_hitbox','no_collision_or_spawn_rejection'],
            'coordinateScaleModels':{'map_coordinates_scaled_profile_radius_unscaled':center_candidates(ps,ts,radius,scale),
                                    'map_coordinates_and_radius_same_scale':center_candidates(ps,ts,radius,1)}})
    report['geometryBlockedBy'] = 'coordinate_scaling_offsets_camera_collision_and_runtime_unverified'
    return report
