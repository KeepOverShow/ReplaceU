"""CPU-backed GPU protocol simulator; never represents real CUDA isolation."""
import json, os, re, sys, time, signal

class SimulatedMemory:
    def __init__(self,limit_mib):
        if limit_mib<=0:raise ValueError('limit must be positive')
        self.limit_mib=limit_mib;self.used_mib=0
    def allocate(self,mib):
        if mib<0:raise ValueError('negative allocation')
        if self.used_mib+mib>self.limit_mib:raise MemoryError('SIMULATED_GPU_MEMORY_LIMIT: requested=%s available=%s MiB'%(mib,self.limit_mib-self.used_mib))
        self.used_mib+=mib
    def release(self,mib):
        if mib<0 or mib>self.used_mib:raise ValueError('invalid release')
        self.used_mib-=mib

def allocation_limits(raw):
    try:
        cards=json.loads(raw)['gpu']
        if len(cards)!=1:raise ValueError('exactly one GPU share required')
        d=cards[0];r=d['resources'];q=str(r['koordinator.sh/gpu-memory'])
        match=re.fullmatch(r'(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti|K|M|G)?',q)
        if not match:raise ValueError('unsupported memory quantity: '+q)
        unit={'Ki':1024,'Mi':1024**2,'Gi':1024**3,'Ti':1024**4,'K':1000,'M':1000**2,'G':1000**3,None:1}[match.group(2)]
        mem=int(float(match.group(1))*unit/(1024**2));cores=int(r['koordinator.sh/gpu-core'])
        if mem<=0 or not 1<=cores<=100:raise ValueError('invalid allocation budget')
        return mem,cores,int(d['minor'])
    except (KeyError,TypeError,IndexError,json.JSONDecodeError) as e:raise ValueError('missing scheduler device allocation') from e

def emit(event,**fields):print(json.dumps(dict(event=event,**fields),ensure_ascii=False),flush=True)
def cpu_matrix(n=32):
    a=[[((i+j)%17)/17 for j in range(n)] for i in range(n)]
    return sum(sum(a[i][k]*a[k][j] for k in range(n)) for i in range(n) for j in range(n))

def run():
    mode=os.getenv('WORKBENCH_MODE','k8s-cpu');duration=int(os.getenv('DURATION_SECONDS','10'));work=os.getenv('WORKLOAD','matmul')
    memory=int(os.getenv('MEMORY_MIB','1024'));cores=int(os.getenv('CORES','25'));minor=None
    if mode in ('k8s-sim','k8s-koord-hami'):
        memory,cores,minor=allocation_limits(os.getenv('DEVICE_ALLOCATION','{}'))
    emit('start',mode=mode,simulated_gpu=mode!='k8s-koord-hami',memory_budget_mib=memory,core_budget=cores,minor=minor)
    if work=='fail':emit('intentional_failure');return 7
    if mode=='k8s-koord-hami':return cuda_run(duration,work,memory)
    ledger=SimulatedMemory(memory)
    if work=='memory':
        try:ledger.allocate(memory+128)
        except MemoryError as e:emit('simulated_limit_rejected',message=str(e),real_cuda_validation=False);return 42
    ledger.allocate(min(256,memory))
    start=time.monotonic();iterations=0;checksum=0
    while time.monotonic()-start<duration:
        before=time.monotonic();checksum=cpu_matrix();iterations+=1
        elapsed=time.monotonic()-before
        # An application-level CPU duty cycle only, NOT a GPU SM enforcement result.
        time.sleep(min(.25,elapsed*(100-cores)/max(1,cores)))
        if iterations%30==0:emit('progress',iterations=iterations,elapsed_seconds=round(time.monotonic()-start,3),virtual_memory_mib=ledger.used_mib,gpu_utilization=None)
    ledger.release(ledger.used_mib);emit('complete',iterations=iterations,elapsed_seconds=round(time.monotonic()-start,3),checksum=checksum,real_cuda_validation=False);return 0

def cuda_run(duration,work,memory):
    import torch
    if not torch.cuda.is_available():raise RuntimeError('CUDA device is unavailable')
    torch.cuda.init()
    if 'libvgpu' not in open('/proc/self/maps').read():raise RuntimeError('HAMi-core mapping not found in workload process')
    emit('cuda_initialized',device=torch.cuda.get_device_name(0),hami_library_mapped=True,isolation_verified=False)
    if work=='memory':
        try:
            x=torch.empty((memory+128)*1024*1024,dtype=torch.uint8,device='cuda');torch.cuda.synchronize()
        except RuntimeError as e:emit('cuda_allocation_rejected',message=str(e));return 42
        emit('isolation_check_failed',reason='allocation above budget succeeded');return 43
    a=torch.rand((512,512),device='cuda');start=time.monotonic();n=0
    while time.monotonic()-start<duration:torch.mm(a,a);torch.cuda.synchronize();n+=1
    emit('complete',iterations=n,elapsed_seconds=time.monotonic()-start,allocated_bytes=torch.cuda.memory_allocated());return 0

if __name__=='__main__':
    try:sys.exit(run())
    except Exception as e:emit('error',type=type(e).__name__,message=str(e));sys.exit(1)
