"""File-first feature readiness and benchmark bindings; no invented difficulty.

Consumes the original archive and captured benchmark definitions. Readiness is
per feature/task: unknown engine geometry never erases explicit configuration
facts or prevents controlled within-family precision comparisons.
"""
import argparse
import collections
import hashlib
import html
import itertools
import json
import math
from pathlib import Path
import zipfile

from intake_sce import mechanism
from sce_extract import fact, raw, split_sections
from sce_geometry import geometry_report
from analyze_geometry import pair_audit, profile_fields

LABELS = {
    'short_transfer': '短距离转移', 'wide_transfer': '大角度转移',
    'micro_adjustment': '微调', 'precision': '精度', 'time_pressure': '时间压力',
    'reflex_window': '限时反应', 'pacing': '节奏递增', 'phased_targets': '分阶段目标',
    'tracking': '追踪', 'projectile': '弹道预判', 'accuracy_constraint': '准确率计分约束',
    'reload_constraint': '击杀补弹／弹匣约束',
}
CHAR_KEYS = ('MainBBType MainBBRadius MainBBHeight ProjBBType ProjBBRadius ProjBBHeight '
             'MaxSpeed Acceleration Gravity MaxHealth HealthRegenPerSec MinRespawnDelay '
             'MaxRespawnDelay SpawnOffsetMin SpawnOffsetMax CameraOffset BlockedSpawnRadius '
             'BlockSpawnFOV BlockSpawnDistance AbilityProfileNames DisableScoring').split()
WEAPON_KEYS = ('Type DamagePerShot TimeBetweenShots MagazineMax AmmoPerShot AmmoReloadedOnKill '
               'ReloadTimeFromEmpty ReloadTimeFromPartial HitscanRadius').split()


def read(path):
    return json.loads(path.read_text())


def write(path, data):
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2, allow_nan=False) + '\n')


def readiness(has_anchors, geometry, diagnostics):
    return {
        'serialized_facts': 'available',
        'declared_multilabel_classification': 'available_with_evidence',
        'conditional_spawn_center_geometry': 'available_under_assumptions' if geometry else 'not_available',
        'benchmark_name_binding': 'candidate' if has_anchors else 'absent',
        'workshop_version_binding': 'not_verified',
        'within_family_precision_order': 'requires_controlled_pair',
        'absolute_angular_features': 'requires_serialization_semantics',
        'fitted_total_difficulty': 'not_fitted',
        'reference_diagnostics': diagnostics,
        'gameplay_observation_required_for_all_tasks': False,
    }


def build(archive, repo, out):
    acquisition = repo / 'docs/research/difficulty-acquisition-2026-10-02/data'
    preparation = repo / 'docs/research/difficulty-preparation-2026-10-02/data'
    targets = {x['scenario_name']: x for x in read(acquisition / 'first-family-targets.json')}
    snapshot = read(preparation / 'benchmark-snapshot.json')
    anchor_index = collections.defaultdict(list)
    for group in snapshot['benchmark_groups']:
        for anchor in group['scenarios']:
            anchor_index[anchor['scenario_name']].append((group, anchor))
    rows, anchors, reports, fields = [], [], {}, {}
    seen = set()
    with zipfile.ZipFile(archive) as z:
        for member in sorted(z.namelist()):
            if not member.lower().endswith('.sce'):
                continue
            content = z.read(member)
            parsed, mech = mechanism(content)
            name = parsed['internal_name']
            if not name or name.casefold() in seen:
                raise ValueError('duplicate or missing internal name')
            seen.add(name.casefold())
            sections, _ = split_sections(content)
            root = sections[0]
            geo = geometry_report(content)
            reports[name], fields[name] = geo, profile_fields(content)
            active = {(p['section'], p['profile_name']) for p in parsed['reachable_profiles']}
            characters, abilities, weapons, bot_refs = [], [], [], []
            for section in sections[1:]:
                kind, pname = section['section'], raw(section, 'Name')
                if (kind, pname) not in active:
                    continue
                if kind == 'Bot Profile':
                    bot_refs.append({'profile': pname, 'character': raw(section, 'CharacterProfile'),
                                     'fields': section['fields']})
                if kind == 'Character Profile':
                    role = 'player' if pname == raw(root, 'PlayerProfile') else 'bot_reachable_role_unresolved'
                    characters.append({'profile': pname, 'role': role,
                                       'fields': [fact(section, k) for k in CHAR_KEYS]})
                if kind == 'Weapon Profile':
                    weapons.append({'profile': pname, 'activation': 'reachable_not_proven_firing',
                                    'fields': [fact(section, k) for k in WEAPON_KEYS]})
                if 'Ability Profile' in kind or kind == 'Bot Rotation Profile':
                    abilities.append({'kind': kind, 'profile': pname, 'line': section['line'],
                                      'fields': section['fields'], 'activation': 'configuration_gated'})
            matches = anchor_index.get(name, [])
            for group, anchor in matches:
                anchors.append({
                    'scenario_name': name, 'file_sha256': parsed['file_sha256'],
                    'game_version': raw(root, 'GameVersion'),
                    'leaderboard_id': anchor['leaderboard_id'],
                    'benchmark_id': group['benchmark_id'], 'benchmark_name': group['benchmark_name'],
                    'tier': group['tier'], 'native_category_path': anchor['native_category_path'],
                    'rank_cutoffs': anchor['rank_cutoffs'],
                    'source_url_template': group['source_url_template'],
                    'retrieved_at': group['retrieved_at'], 'threshold_snapshot_sha256': group['response_sha256'],
                    'binding_status': 'exact_internal_name_candidate_version_not_verified',
                    'cross_benchmark_rank_equivalence_verified': False,
                    'interpretation': 'native performance thresholds; not a numeric scene difficulty label',
                })
            target = targets.get(name, {})
            # The inherited family list is provisional. Pressure and vibration
            # remain separate mechanism strata instead of one wall-reload model.
            original_family = target.get('family_candidate')
            if original_family and 'time_pressure' in mech['tags']:
                stratum = original_family + ':pressure'
            elif original_family and 'vibrat' in (raw(root, 'Description') or '').lower():
                stratum = original_family + ':vibration'
            else:
                stratum = original_family
            rows.append({
                'name': name, 'archive_member': member, 'file_sha256': parsed['file_sha256'],
                'game_version': raw(root, 'GameVersion'),
                'family_candidate': original_family, 'mechanism_stratum_candidate': stratum,
                'family_evidence': 'existing benchmark sampling registry; provisional' if original_family else 'unknown',
                'leakage_group_candidate': target.get('leakage_group_candidate'),
                'native_categories': sorted({' / '.join(a['native_category_path']) for _, a in matches}),
                'declared_skill': mech['declaredSkill'] or None,
                'tags': mech['tags'], 'tag_evidence': mech['evidence'],
                'root_facts': parsed['root_fields'], 'initial_bot_slot_count': parsed['initial_bot_slot_count'],
                'slot_count_interpretation': 'initial configured slots; not visible or scoreable target count',
                'bot_profiles': bot_refs, 'character_profiles': characters,
                'reachable_weapons': weapons, 'player_weapon_facts': mech['playerWeaponFacts'],
                'abilities_and_rotations': abilities,
                'map_facts': geo['map'], 'conditional_geometry': geo['geometryCandidates'],
                'geometry_missing_semantics': geo.get('geometryBlockedBy'),
                'authoritative_angular_features': None,
                'readiness': readiness(bool(matches), bool(geo['geometryCandidates']), parsed['reference_diagnostics']),
            })
    families = collections.defaultdict(list)
    for row in rows:
        if row['family_candidate']:
            families[row['family_candidate']].append(row['name'])
    pairs = [pair_audit(a, b, reports, fields) for members in families.values()
             for a, b in itertools.combinations(members, 2)]
    controls = [p for p in pairs if p['sizeOnlyConfigurationPair']]
    for p in controls:
        p['radius_ratios_b_over_a'] = []
        for c in p['sizeChanges']:
            if c['path'][-1] == 'MainBBRadius':
                a, b = float(c['before'][0]), float(c['after'][0])
                if a > 0 and b > 0:
                    p['radius_ratios_b_over_a'].append({'profile': c['path'][1], 'ratio': b/a,
                        'relative_log_inverse_radius_b_minus_a': math.log2(a/b),
                        'unit': 'dimensionless configuration contrast; not difficulty points'})
        p['usable_for'] = 'within_family_precision_feature_order_under_fixed_geometry_semantics'
    controlled_names = {p[k] for p in controls for k in ['a', 'b']}
    for row in rows:
        if row['name'] in controlled_names:
            row['readiness']['within_family_precision_order'] = 'available_conditionally'
    anchored_names = {a['scenario_name'] for a in anchors}
    unanchored = [r['name'] for r in rows if r['name'] not in anchored_names]
    tag_counts = dict(collections.Counter(t for r in rows for t in r['tags']))
    family_stats = [{'family_candidate': f, 'scenario_count': len(ns),
                     'mechanism_strata': sorted({r['mechanism_stratum_candidate'] for r in rows
                                                 if r['name'] in ns and r['mechanism_stratum_candidate']}),
                     'benchmark_groups': sorted({a['benchmark_id'] for a in anchors if a['scenario_name'] in ns}),
                     'controlled_size_only_pairs': sum(p['a'] in ns and p['b'] in ns for p in controls)}
                    for f, ns in sorted(families.items())]
    gaps = {
        'do_now_without_gameplay': ['use file facts and controlled precision order',
            'resolve map scale/team/default/offset semantics from documentation or source',
            'model ability activation predicates and rotation phases from files',
            'fit and validate within-family ordinal baselines after specifying labels and features'],
        'optional_independent_validation': 'editor/gameplay observations only for specific unresolved semantics; not mandatory for all scenes',
        'without_benchmark_anchor': unanchored,
        'without_supported_tags': [r['name'] for r in rows if not r['tags']],
        'without_registered_family': [r['name'] for r in rows if not r['family_candidate']],
        'conditional_geometry_count': sum(bool(r['conditional_geometry']) for r in rows),
        'absolute_geometry_missing_count': len(rows),
        'version_binding_missing_count': len(anchored_names),
        'family_coverage': family_stats,
        'model_identifiability': 'three ordered tiers per family cannot identify many feature weights; begin with few features and controlled comparisons',
        'scope': 'first 30 targets cover clicking/flick candidates; uploaded extra exercises do not establish multi-tier tracking/switching coverage',
        'cross_family_scale': 'requires defensible bridge evidence; raw rank thresholds and playlist tiers alone do not define equal difficulty',
        'user_material_required_now': [],
        'sampling_decision': 'finish current-family baseline before requesting additional files',
    }
    summary = {'archive_sha256': hashlib.sha256(archive.read_bytes()).hexdigest(),
        'base_commit': 'abc8532520be1ebfecc554c79da7d00c02449240',
        'scenario_count': len(rows), 'benchmark_anchor_rows': len(anchors),
        'anchored_scenarios': len(anchored_names), 'tag_counts': tag_counts,
        'size_only_pairs': len(controls), 'controlled_scenarios': len(controlled_names),
        'conditional_geometry_scenarios': gaps['conditional_geometry_count'],
        'calibrated_total_difficulty_models': 0,
        'benchmark_evidence': 'captured definitions with retained retrieval timestamps; not live refreshed',
        'runtime_is_global_prerequisite': False}
    out.mkdir(parents=True, exist_ok=True)
    write(out/'scenario-features.json', {'summary': summary, 'scenarios': rows})
    write(out/'benchmark-anchors.json', {'summary': summary, 'anchors': anchors})
    write(out/'feature-gaps.json', gaps)
    write(out/'controlled-precision-pairs.json', {'summary': summary, 'pairs': controls})
    render(out/'report.html', summary, rows, anchors, gaps, controls)
    return summary


def esc(x):
    return html.escape(str(x))


def table(headers, rows):
    return '<table><thead><tr>'+''.join('<th>'+esc(h)+'</th>' for h in headers)+'</tr></thead><tbody>'+''.join(
        '<tr>'+''.join('<td>'+esc(c)+'</td>' for c in row)+'</tr>' for row in rows)+'</tbody></table>'


def render(path, s, rows, anchors, gaps, controls):
    parts = ['<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>SCE 分类、特征与难度锚点</title>',
        '<style>body{font:16px/1.65 system-ui,sans-serif;color:#202a35;max-width:1450px;margin:40px auto;padding:0 24px;background:#f6f8fa}h1,h2{color:#10375a}table{border-collapse:collapse;background:white;width:100%;margin:18px 0;font-size:14px}th,td{border:1px solid #d6dfe8;padding:9px;text-align:left;vertical-align:top}th{background:#e5eef7}pre{white-space:pre-wrap;overflow-wrap:anywhere;font-size:12px}details{background:white;border:1px solid #d6dfe8;padding:12px;margin:8px 0}.note{background:#e5eef7;padding:16px}input{font:inherit;padding:8px;width:320px;max-width:90%}</style>',
        '<h1>SCE 分类、特征与 benchmark 锚点</h1><p>2026-10-02 · 文件优先研究 · 基于提交 abc8532 与用户上传快照</p>',
        f'<p class="note">已核查 {s["scenario_count"]} 个场景；{s["anchored_scenarios"]} 个场景匹配到 {s["benchmark_anchor_rows"]} 条原生 benchmark 归属。'
        f'获得 {s["size_only_pairs"]} 对仅尺寸变化的配置对照、{s["conditional_geometry_scenarios"]} 个场景的条件几何。完整难度模型尚未拟合。</p>',
        '<p>文件明确值可立即使用；条件特征保留假设；缺失字段保留未知。游戏内观察只用于有具体缺口的验证，不作为全部分析的门槛。当前文件名匹配未证明 Workshop 内容版本相同。</p>',
        '<h2>1. 细分类与配置特征（58 场景）</h2><p>多标签；作者说明与字段证据分别保存。短／大角度标签目前主要来自作者说明，未冒充已计算角距离。未标标签不表示该机制不存在。原始速度／尺寸不跨地图比较。</p>',
        '<input id="search" placeholder="搜索场景、标签或家族" aria-label="筛选场景">',
        table(['场景','候选家族／机制层','多标签','初始槽位','bot 可达角色尺寸／速度（配置单位）','地图','条件几何'], [
            [r['name'],r['mechanism_stratum_candidate'] or '未知', '、'.join(LABELS[t] for t in r['tags']) or '未知',
             r['initial_bot_slot_count'], '; '.join(p['profile']+': '+', '.join(f["key"]+'='+str(f.get('parsed_value'))
                for f in p['fields'] if f['key'] in ['MainBBRadius','MainBBHeight','MaxSpeed'] and f['status']=='explicit')
                for p in r['character_profiles'] if p['role']!='player'),r['map_facts']['format'],
             '有；需按假设解释' if r['conditional_geometry'] else '尚无'] for r in rows]),
        '<h2>2. 原生 benchmark 锚点</h2><p>保留既有采集时间与响应哈希。门槛是该场景的成绩要求，不是跨场景难度分；段位同名也不代表跨 benchmark 等难度。</p>',
        table(['场景','benchmark / 档位','原生分类','排行榜 ID','原生分数门槛','采集时间'], [
            [a['scenario_name'],a['benchmark_name']+' / '+a['tier'],' / '.join(a['native_category_path']),
             a['leaderboard_id'],'; '.join(f'{k}: {v}' for k,v in a['rank_cutoffs'].items()),a['retrieved_at']] for a in anchors]),
        '<h2>3. 可立即使用的精度配置对照</h2><p>地图及其余序列化配置相同，只改变命中盒尺寸；支持固定其余语义条件下的精度需求方向。八对来自四个家族，不能当作八个独立实验，也不能当作总难度预测。比值只描述配置半径变化。</p>',
        table(['A','B','B/A 主碰撞盒半径','精度配置方向'], [[p['a'],p['b'],
            '; '.join(x['profile']+': '+format(x['ratio'],'.4g') for x in p['radius_ratios_b_over_a']),
            'B 更小' if p['mainHitboxPrecisionDemandUnderFixedRuntime']=='b_smaller' else p['mainHitboxPrecisionDemandUnderFixedRuntime']] for p in controls]),
        '<h2>4. 缺口与下一步</h2><p>暂时无需补交文件。先使用同家族对照建立简单精度基线，再处理生成约束、阶段与压力门控；模型验证留出完整家族与版本。</p>',
        table(['项目','当前结果／处理'],[
            ['缺少原生 benchmark 锚点',str(len(gaps['without_benchmark_anchor']))+' 个；保持未标定，不从名称或 playlist 档位补标签'],
            ['无支持标签',str(len(gaps['without_supported_tags']))+' 个；核查作者说明与配置规则'],
            ['条件几何',str(gaps['conditional_geometry_count'])+' 个；确认缩放、生成偏移、视点与筛选语义'],
            ['完整角度特征','58 个尚未确认；可先做不依赖绝对角度的同家族配置对照'],
            ['完整难度函数','0 个；少量档位不能识别大量权重'],
            ['追踪／切换覆盖','额外上传场景不等于完整分档样本；先完成当前点击家族基线'],
            ['游戏内验证','仅对文档、源码与文件分析无法消除的具体歧义使用'],
        ]),'<details><summary>无 benchmark 锚点的场景</summary><pre>'+esc('\n'.join(gaps['without_benchmark_anchor']))+'</pre></details>',
        '<h2>5. 逐场景证据与特征状态</h2>']
    for r in rows:
        parts.append('<details><summary>'+esc(r['name'])+'</summary><pre>'+esc(json.dumps(r,ensure_ascii=False,indent=2))+'</pre></details>')
    parts.append('<script>document.getElementById("search").addEventListener("input",e=>{let q=e.target.value.toLowerCase();document.querySelectorAll("table")[0].querySelectorAll("tbody tr").forEach(r=>r.hidden=!r.textContent.toLowerCase().includes(q));});</script></html>')
    path.write_text('\n'.join(parts))


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument('archive', type=Path)
    ap.add_argument('--repo', type=Path, required=True)
    ap.add_argument('--output', type=Path, required=True)
    args = ap.parse_args()
    print(json.dumps(build(args.archive, args.repo, args.output), ensure_ascii=False))
