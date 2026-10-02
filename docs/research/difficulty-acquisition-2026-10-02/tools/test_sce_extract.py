import unittest

from sce_extract import extract, split_sections
from capture_vdim import capture_document, secondary_rows
from capture_workshop import WorkshopItems
from intake_sce import mechanism


class ExtractionTests(unittest.TestCase):
    def test_abilities_prevent_stationary_assumption_and_case_is_reported(self):
        result = extract(b'Name=T\nPlayerProfile=P\nAddedBots=target.bot\n[Character Profile]\nName=P\n[Bot Profile]\nName=Target\nCharacterProfile=T\n[Character Profile]\nName=T\nMaxSpeed=0\nAbilityProfileNames=Approach.abilmov\n[Movement Ability Profile]\nName=Approach\nMainVelocity=4000\n')
        self.assertTrue(any(d['status'] == 'case_variant_reference' for d in result['reference_diagnostics']))
        self.assertTrue(any(p['section'] == 'Movement Ability Profile' for p in result['reachable_profiles']))
        self.assertFalse(result['full_mechanics_verified'])

    def test_intake_mixed_labels_and_unknown_without_title_guess(self):
        _, m = mechanism(b'Name=Pressure Wide Hard\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\n')
        self.assertEqual(m['tags'], [])
        _, m = mechanism(b'Name=T\nDescription=After killing 3 big bots, force wideflicks and micros\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\n')
        self.assertEqual(m['tags'], ['micro_adjustment', 'phased_targets', 'wide_transfer'])
        self.assertIsNone(m['transitionAngle'])
        self.assertEqual(m['role'], 'unknown')

    def test_only_root_reachable_profiles_are_used(self):
        content = b'Name=Test\nPlayerProfile=P\nAddedBots=target.bot;target.bot\n[Character Profile]\nName=P\nWeaponProfileNames=Gun;;;;;;;\n[Weapon Profile]\nName=Gun\nType=Hitscan\n[Bot Profile]\nName=unused\nCharacterProfile=missing\n[Bot Profile]\nName=target\nCharacterProfile=T\n[Character Profile]\nName=T\nMainBBRadius=2\nProjBBRadius=50\nMaxSpeed=0\n'
        result = extract(content)
        self.assertEqual(result['initial_bot_slot_count'], 2)
        self.assertEqual(result['all_profile_counts']['Bot Profile'], 2)
        self.assertEqual(result['reference_diagnostics'], [])
        self.assertNotIn('unused', [p['profile_name'] for p in result['reachable_profiles']])
        target = next(p for p in result['reachable_profiles'] if p['profile_name'] == 'T')
        facts = {f['key']: f for f in target['fields']}
        self.assertEqual(facts['MaxSpeed']['parsed_value'], 0)
        self.assertEqual(facts['MaxHealth']['status'], 'missing')
        self.assertEqual(facts['MainBBRadius']['parsed_value'], 2)
        self.assertEqual(facts['ProjBBRadius']['parsed_value'], 50)
        self.assertIsNone(result['geometry_features']['angular_size'])
        self.assertFalse(result['calibration_eligible'])

    def test_duplicate_sections_are_not_overwritten(self):
        result = extract(b'Name=T\nPlayerProfile=P\nAddedBots=a.bot\n[Character Profile]\nName=P\n[Bot Profile]\nName=a\n[Bot Profile]\nName=a\n')
        self.assertTrue(any(d['status'] == 'ambiguous_profile_name' for d in result['reference_diagnostics']))

    def test_duplicate_root_key_does_not_silently_pick_last(self):
        result = extract(b'Name=A\nName=B\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\n')
        self.assertIsNone(result['internal_name'])
        self.assertEqual(result['root_fields'][0]['status'], 'ambiguous_duplicate_key')

    def test_rotation_cycle_and_empty_slots_are_preserved(self):
        result = extract(b'Name=T\nPlayerProfile=P\nAddedBots=a.rot;;\n[Character Profile]\nName=P\n[Bot Rotation Profile]\nName=a\nProfileNames=b.rot\n[Bot Rotation Profile]\nName=b\nProfileNames=a.rot\n')
        self.assertEqual(result['added_bot_slots'], ['a.rot', '', ''])
        self.assertTrue(any(d['status'] == 'cyclic_reference' for d in result['reference_diagnostics']))

    def test_map_data_is_opaque_and_missing_is_not_zero(self):
        sections, data = split_sections(b'Name=T\n[Map Data]\nvertices=123\n[Character Profile]\nName=not_a_profile\n')
        self.assertEqual(len(sections), 2)
        self.assertIn('[Character Profile]', data)
        result = extract(b'Name=T\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\nMainBBRadius=nan\n')
        radius = next(f for f in result['reachable_profiles'][0]['fields'] if f['key'] == 'MainBBRadius')
        self.assertEqual(radius['status'], 'invalid')

    def test_workshop_parser_keeps_exact_titles_and_ids(self):
        parser = WorkshopItems()
        parser.feed('<a href="https://steamcommunity.com/sharedfiles/filedetails/?id=123"><img alt="Map A"/></a><a href="/sharedfiles/filedetails/?id=456"><img alt="Map A Hard"/></a><img alt="Unrelated"/>')
        self.assertEqual(parser.items, [{'publishedfileid': '123', 'title': 'Map A'}, {'publishedfileid': '456', 'title': 'Map A Hard'}])

    def test_vdim_catalog_keeps_player_tier_separate(self):
        paragraphs = []
        for day, category, part in [('MONDAY', 'CLICKING', 'I'), ('TUESDAY', 'CLICKING', 'II'), ('WEDNESDAY', 'TRACKING', 'I'), ('THURSDAY', 'TRACKING', 'II'), ('FRIDAY', 'SWITCHING', 'I'), ('SATURDAY', 'SWITCHING', 'II')]:
            paragraphs.append(f'<p>{day}</p>')
            for tier in ['ENTRY', 'NOVICE', 'ADEPT', 'INTERMEDIATE', 'ADVANCED', 'ELITE']:
                paragraphs.extend([f'<p>&gt; PLAY {tier} {category} {part}</p>', f'<p>KovaaKs{day}{tier}</p>'])
        rows = capture_document(''.join(paragraphs).encode())
        self.assertEqual(len(rows), 36)
        self.assertTrue(all(not r['scenario_difficulty_inheritance_allowed'] for r in rows))
        self.assertTrue(all(not r['warmup_roles_verified'] for r in rows))


if __name__ == '__main__':
    unittest.main()
