import json
import math
import unittest

from sce_geometry import parse_map, eligible_json_spawns, center_candidates, geometry_report
from analyze_geometry import pair_audit, profile_fields


class GeometryTests(unittest.TestCase):
    def test_json_profile_and_team_filters_are_explicit(self):
        objs = []
        for team, profile in [(1,''),(2,'Small,Large'),(2,'Helper')]:
            objs.append({'type':'gameObject','name':'SpawnPoint','location':'100, 0, 0','properties':[
                {'name':'TeamMask','value':team},{'name':'PermittedCharacterProfiles','value':profile},{'name':'Weight','value':1}]})
        m = parse_map(json.dumps({'objects':objs}))
        self.assertEqual(len(eligible_json_spawns(m,'2','Small')),1)
        self.assertEqual(len(eligible_json_spawns(m,'1','Player')),1)
        self.assertIsNone(eligible_json_spawns(m,None,'Player'))
        self.assertEqual(m['spawnCount'],3)
        self.assertEqual(m['uniquePositionCount'],1) # multiplicity not lost

    def test_legacy_omissions_are_not_team_defaults(self):
        m = parse_map('reflex map version 8\nglobal\n\tentity\n\t\ttype PlayerSpawn\n\t\tVector3 position 1 2 3\n\t\tBool8 teamA 0\n\tbrush\n\t\tvertices\n\t\t\t9 9 9\n')
        self.assertEqual(m['spawnCount'],1)
        self.assertNotIn('teamB',m['spawns'][0]['properties'])
        self.assertIsNone(eligible_json_spawns(m,'2','T'))

    def test_malformed_and_duplicate_data_are_rejected(self):
        self.assertEqual(parse_map('{"objects":[],"objects":[]}')['status'],'invalid')
        self.assertIsNone(center_candidates([[0,0,0]],[[0,0,0]],1,1))
        self.assertEqual(parse_map('unknown')['status'],'unsupported')

    def test_sphere_and_pair_angles_with_scale_sensitivity(self):
        first = center_candidates([[0,0,0]],[[10,0,0],[0,10,0]],1,1)
        second = center_candidates([[0,0,0]],[[10,0,0],[0,10,0]],1,2)
        self.assertAlmostEqual(first['angularDiameterDegrees']['min'],math.degrees(2*math.asin(.1)))
        self.assertAlmostEqual(first['spawnCenterPairAngleDegrees']['max'],90)
        self.assertAlmostEqual(second['spawnCenterPairAngleDegrees']['max'],90)
        self.assertLess(second['angularDiameterDegrees']['max'],first['angularDiameterDegrees']['max'])

    def test_missing_properties_and_default_differences_block_claims(self):
        a = b'Name=A\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\nMainBBRadius=1\n'
        b = b'Name=B\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\nMainBBRadius=2\nMaxSpeed=0\n'
        reports = {n:{'map':{'semanticSHA256':'same'}} for n in ['A','B']}
        p = pair_audit('A','B',reports,{'A':profile_fields(a),'B':profile_fields(b)})
        self.assertFalse(p['sizeOnlyConfigurationPair'])
        self.assertEqual(p['otherChangedFieldCount'],1)
        self.assertIsNone(geometry_report(a)['authoritativeGeometry'])
        c = b'Name=C\nDifficultyTag=3\nPlayerProfile=P\nAddedBots=\n[Character Profile]\nName=P\nMainBBRadius=2\n'
        q = pair_audit('A','C',{'A':reports['A'],'C':reports['B']},{'A':profile_fields(a),'C':profile_fields(c)})
        self.assertTrue(q['sizeOnlyConfigurationPair'])
        self.assertFalse(q['empiricalDifficultyOrderVerified'])


if __name__=='__main__':
    unittest.main()
