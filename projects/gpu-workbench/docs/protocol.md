# 调度协议与模拟边界

## 运行中的真实部分

Go HTTP API → SQLite 即时事务 → Controller → Kubernetes Job → Koordinator DeviceShare → Pod 设备分配注解 → kubelet → Python CPU 计算 → Job 终态 → 释放团队额度。

## 被替换的硬件部分

1. `scripts/sim-devices.py` 代替真实 GPU 采集，发布两个带 SIM-GPU 标识的 Device CR。
2. `workloads/worker.py` 从 Pod 的真实设备分配注解读取预算。
3. SimulatedMemory 用整数账本模拟显存分配，超限时抛出错误；不分配同等数量的宿主机内存。
4. CPU 计算与休眠按比例交替，只演示应用层占空比，不能代表 GPU SM 利用率或 HAMi-core 的算力限制精度。
5. 模拟模式不设置 HAMi-core provider 标签，不挂载 libvgpu.so，不伪造 CUDA 调用结果。

## 真实硬件迁移

使用独立数据目录运行 `k8s-koord-hami`，连接专用 Linux NVIDIA 测试节点。启用 GPUEnvInject / NRI，部署 hami-daemon 与驱动、容器运行时；用真实采集替换模拟 Device CR。业务请求保持 gpu.shared / gpu-core / gpu-memory 协议。

先检查 CUDA 基础调用，再检查设备分配、环境变量、库挂载、应用进程映射，最后执行超限与邻居干扰对照。标签和注解只能证明配置/分配，不能证明隔离效果。

## 取消与一致性

- 请求先事务性记录 Cancelling。
- 协调器请求暂停 Job；只有 JobSuspended=True（或已有真实终态条件）且无活动 Pod，才释放额度。spec.suspend=true 本身只是意图，不能证明控制器已停止创建 Pod。
- 缺失 Job 时创建同名暂停对象作为取消标记；迟到的同名创建不能覆盖它。
- 对象归属、请求摘要或 UID 冲突时停止推进并保留额度。
- 数据库绑定模式及命名空间 UID；不能把旧账本直接切换到另一个集群。
- 仅支持单个控制进程，文件锁拒绝重复实例；不宣称高可用。

默认保留 Job/Pod 与数据库终态用于学习。清理前必须停止平台、确认无活动任务并保存证据；删除取消标记会丢失其防迟到创建作用。不要删除运行中的 Job 来代替取消 API。
