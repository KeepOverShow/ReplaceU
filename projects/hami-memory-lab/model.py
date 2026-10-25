"""Deterministic teaching traces; neither CUDA nor HAMi-core is executed."""
import json
from pathlib import Path

class Lab:
    def __init__(self, limit=4096, used=0):
        self.limit, self.used, self.calls = limit, used, 0
        self.free_calls = 0
        self.trace = []
    def event(self, owner, action, **extra):
        self.trace.append(dict(owner=owner, action=action, used_mib=self.used,
                               driver_alloc_calls=self.calls, driver_free_calls=self.free_calls, scheduler_rpc=0, **extra))
    def alloc(self, amount, driver_ok=True):
        self.event('应用 → HAMi-core', '申请 %s MiB；读取逻辑预算与共享账本' % amount)
        if self.used + amount > self.limit:
            self.event('HAMi-core', '逻辑预算拒绝；不调用真实分配函数；不登记新分配',
                       result='CUDA_ERROR_OUT_OF_MEMORY', failure_origin='budget')
            return False
        self.event('HAMi-core → Driver', '预算通过，只取得继续调用的资格')
        self.calls += 1
        if not driver_ok:
            self.event('Driver → HAMi-core → 应用', '教学桩强制返回物理层失败；成功记账路径未执行',
                       result='CUDA_ERROR_OUT_OF_MEMORY', failure_origin='driver')
            return False
        self.used += amount
        self.event('HAMi-core → 共享账本', 'Driver 成功后登记地址与使用量；向应用返回成功',
                   result='CUDA_SUCCESS', failure_origin=None)
        return True
    def free(self, amount):
        assert 0 < amount <= self.used
        self.event('应用 → HAMi-core → Driver', '释放已跟踪的 %s MiB；本例假定底层释放成功' % amount)
        self.free_calls += 1
        self.used -= amount
        self.event('HAMi-core → 共享账本', '撤销该分配记账；Pod 的调度预算不因此自动归还')

def scenarios():
    result = {}
    for key, title, used, amount, ok in [
        ('success', '预算内且底层成功', 2048, 1024, True),
        ('budget', '预算侧拒绝', 2048, 3072, True),
        ('driver', '预算内但底层失败', 2048, 1024, False),
    ]:
        lab=Lab(used=used);lab.alloc(amount,ok)
        result[key]={'title':title,'simulated':True,'trace':lab.trace}
    lab=Lab(used=2048);lab.free(2048);lab.alloc(3072)
    result['release']={'title':'释放后重新申请','simulated':True,'trace':lab.trace}
    # A forced interleaving, not an executed upstream concurrency test.
    trace=[
        dict(owner='进程 A / B',action='共享初值 2 GiB，各申请 1.5 GiB；A、B 都在对方记账前读取 2 GiB',used_mib=2048,driver_alloc_calls=0,driver_free_calls=0,scheduler_rpc=0),
        dict(owner='两个检查窗口',action='两边各自看到 2 + 1.5 ≤ 4；单次读和写加锁不等于整段原子',used_mib=2048,driver_alloc_calls=0,driver_free_calls=0,scheduler_rpc=0),
        dict(owner='Driver 与共享账本',action='假定两次底层调用都成功，随后各加 1.5 GiB，合计达到 5 GiB',used_mib=5120,driver_alloc_calls=2,driver_free_calls=0,scheduler_rpc=0),
    ]
    result['race']={'title':'跨进程交错反例','simulated':True,'trace':trace}
    return result

if __name__=='__main__':
    data=json.dumps(scenarios(),ensure_ascii=False,indent=2)+'\n'
    print(data,end='')
