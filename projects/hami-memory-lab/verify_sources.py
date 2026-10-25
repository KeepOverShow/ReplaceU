"""Verify inspected committed bytes, not deployment behavior."""
import argparse, hashlib, json, subprocess
from pathlib import Path

def git(root, *args):
    return subprocess.check_output(['git','-C',str(root),*args])

def verify(hami, core):
    lock=json.loads(Path(__file__).with_name('source-lock.json').read_text())
    for name,root in [('hami',hami),('hami_core',core)]:
        item=lock[name]
        for path, expected in item['sha256'].items():
            data=git(root,'show',item['revision']+':'+path)
            if hashlib.sha256(data).hexdigest()!=expected:
                raise ValueError('source hash mismatch: '+name+'/'+path)
        print(name+': pinned committed source hashes match')
    tree=git(hami,'ls-tree',lock['hami']['revision'],'libvgpu').decode().split()
    if tree[:2]!=['160000','commit'] or tree[2]!=lock['hami_core']['revision']:
        raise ValueError('libvgpu gitlink mismatch')
    print('HAMi libvgpu gitlink matches the selected HAMi-core revision')
    print('This verifies source identity only; CUDA validation remains not_run.')

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--hami',required=True);p.add_argument('--core',required=True)
    args=p.parse_args();verify(args.hami,args.core)
