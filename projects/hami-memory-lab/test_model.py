import unittest
from model import Lab, scenarios

class Contracts(unittest.TestCase):
    def test_budget_rejection_never_calls_allocator(self):
        lab=Lab(used=2048)
        self.assertFalse(lab.alloc(3072))
        self.assertEqual((lab.used,lab.calls),(2048,0))
    def test_equal_oom_errors_do_not_identify_the_cause(self):
        a,b=Lab(used=2048),Lab(used=2048)
        a.alloc(3072);b.alloc(1024,False)
        self.assertEqual(a.trace[-1]['result'],b.trace[-1]['result'])
        self.assertNotEqual(a.trace[-1]['failure_origin'],b.trace[-1]['failure_origin'])
        self.assertEqual((b.used,b.calls),(2048,1))
    def test_free_changes_runtime_usage_then_reallocation_succeeds(self):
        lab=Lab(used=2048);lab.free(2048)
        self.assertTrue(lab.alloc(3072))
        self.assertEqual(lab.used,3072)
    def test_success_and_failure_leave_scheduler_out_of_runtime_path(self):
        for case in scenarios().values():
            self.assertTrue(all(e['scheduler_rpc']==0 for e in case['trace']))
    def test_forced_interleaving_is_counterexample_to_atomicity(self):
        self.assertGreater(scenarios()['race']['trace'][-1]['used_mib'],4096)

if __name__=='__main__':unittest.main()
