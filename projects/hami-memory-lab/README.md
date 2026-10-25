# HAMi 显存机制实验

第二课配套，分开保存可运行模型和待 NVIDIA 硬件验证的探针。模型不执行 HAMi-core；CPU 单测不能证明 CUDA 限制有效。

## 普通 CPU 环境可复现

```sh
python3 -m unittest discover -s projects/hami-memory-lab -v
python3 projects/hami-memory-lab/model.py > /tmp/hami-cases.json
diff projects/hami-memory-lab/cases.json /tmp/hami-cases.json
```

五组轨迹：预算内成功、预算拒绝（不发起真实显存分配）、底层失败、释放再申请、强制跨进程交错。最后一组是“检查与更新分离”的反例，不是运行了上游并发测试。模型忽略 context 开销、设备编号映射、初始偏移、池和异步等因素；真实 usage 未必等于示例申请之和。

`cases.json` 是网页逐步实验的数据源，由模型生成；修改模型后须重新生成并构建站点。

## 固定源码基线

- HAMi v2.7.0：`f754a8c4621d969aba4c7c10b1d1d6d41fe6b530`
- 其 `libvgpu` gitlink：`3506e6222ca0f235cb745bc805832f422c484c17`
- 本课主线按第二个提交的同步 `cuMemAlloc_v2` 路径解释。
- 独立比较提交 `cdb5260760f16221c027ffeedb747c38a2f4deb5` 已有共享预留与失败回滚路径；它不属于上述 v2.7.0 基线。不要把两版实现混作部署保证。

用 `source-lock.json` 和两张图的 `candidate.json` 查看完整路径及行号。源码克隆后 checkout 到锁定提交，再比较 `git ls-tree HEAD libvgpu`；运行环境还须补驱动、GPU、CUDA、容器运行时与实际镜像摘要。

## NVIDIA 环境中的探针（当前未编译、未运行）

`probe.c` 创建 context，分配并 touch 两份内存，释放后重试第二份。逐步输出 Driver 返回码和错误名。它不自动宣称通过；退出 0 仅表示走完报告流程，必须检查每行结果及错误来源。

```sh
CUDA_HOME=/usr/local/cuda bash projects/hami-memory-lab/build-probe.sh
# 在专用 CUDA 测试容器中，完成驱动与设备暴露后：
projects/hami-memory-lab/.runtime/probe 256 384 > control.jsonl
# 使用本课匹配构建的库，并按实际挂载情况设置绝对路径；
# 缓存路径必须可写，每轮独立，使用已准备好的 /tmp/vgpulock。
LD_PRELOAD=/absolute/path/libvgpu.so \
CUDA_DEVICE_MEMORY_LIMIT=512m \
CUDA_DEVICE_MEMORY_SHARED_CACHE=/tmp/hami-probe-unique.cache \
projects/hami-memory-lab/.runtime/probe 256 384 > limited.jsonl
```

控制组必须确认没有其他机制通过 `/etc/ld.so.preload` 等方式加载 HAMi-core；不能只删 LD_PRELOAD 就认定“未加载”。512 MiB 内还可能包含上下文等开销，必须确认首次分配能成功，再分析超限与释放重试；否则结果不确定。

真正验收需记录：相同 GPU UUID、驱动和镜像；进程实际加载的库；每设备限额与共享缓存身份；物理余量及其他工作负载；Hook 日志与 Driver 原始返回。`nvidia-smi` 的展示可能也受 Hook 影响，应从未加载库的宿主机路径独立观测，并解释不同统计口径。

预期只有在足够物理余量、首次成功且受支持路径命中的前提下，受限组第二次超额申请才应由预算侧拒绝；仅凭 OOM 名字不能判定。释放后重试、异步 API、并发、异常退出与陈旧账本各有独立验证要求。此探针没有覆盖所有这些场景。

## 证据状态

- 已执行：Python 模型单测与轨迹重建核对。
- 已阅读：固定提交的配置交付、同步检查、分配、记账和释放源码。
- 待硬件验证：探针编译、真实 CUDA 路径、并发与崩溃恢复。
- 模型轨迹：解释原理；不是运行日志。

独立核对源码快照（需要 git，输出仅验证身份，不验证运行正确性）：

```sh
git clone https://github.com/Project-HAMi/HAMi.git /tmp/hami-read
git clone https://github.com/Project-HAMi/HAMi-core.git /tmp/hami-core-read
python3 projects/hami-memory-lab/verify_sources.py --hami /tmp/hami-read --core /tmp/hami-core-read
```

校验器读取锁定提交的已提交字节并核对 SHA-256 和 gitlink；它不拿工作区的未提交修改作为该提交的证据。
