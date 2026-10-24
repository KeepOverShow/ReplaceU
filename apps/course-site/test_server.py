import unittest
from server import allocate,schedule
class Cases(unittest.TestCase):
 def test_rejection_keeps_budget_and_skips_driver(self):
  r=allocate(2048,3072);self.assertFalse(r['accepted']);self.assertEqual(r['after_mib'],2048);self.assertFalse(r['driver_called'])
 def test_exact_boundary(self):self.assertTrue(allocate(2048,2048)['accepted'])
 def test_forged_usage_rejected(self):
  for v in (-1,4097,True):
   with self.assertRaises(ValueError):allocate(v,2048)
 def test_total_free_does_not_imply_single_card_fit(self):
  r=schedule('fragmented');self.assertGreater(r['total_free_mib'],r['request_mib']);self.assertEqual(r['state'],'Pending')
 def test_pack_and_share(self):
  self.assertEqual(schedule('packed')['selected_minor'],1);self.assertEqual(schedule('shared')['selected_minor'],0)
if __name__=='__main__':unittest.main()
