import unittest
from worker import SimulatedMemory, allocation_limits
class TestMemory(unittest.TestCase):
    def test_enforces_budget_without_allocating_host_ram(self):
        m=SimulatedMemory(1024);m.allocate(768)
        with self.assertRaises(MemoryError):m.allocate(512)
        self.assertEqual(m.used_mib,768);m.release(768);self.assertEqual(m.used_mib,0)
    def test_reads_scheduler_allocation(self):
        self.assertEqual(allocation_limits('{"gpu":[{"minor":1,"resources":{"koordinator.sh/gpu-memory":"2Gi","koordinator.sh/gpu-core":"25"}}]}'),(2048,25,1))
        with self.assertRaises(ValueError):allocation_limits('{}')
if __name__=='__main__':unittest.main()
