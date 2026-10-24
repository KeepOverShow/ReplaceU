# 真实 GPU 接入路径（尚未在本机验证）

本机使用 `k8s-sim`。以下为后续 Linux NVIDIA 节点的接入说明，不代表已完成真实硬件验证。

1. 使用专用测试节点，检查 NVIDIA 驱动、容器工具链与 CUDA 基础任务；确认 containerd >= 1.7，并启用 NRI。
2. 采用已核对兼容性的 Koordinator 版本。使用真实设备报告器，不运行 sim-devices.py，不手工维护真实 GPU 的 Device CR。
3. 启用 koordlet `GPUEnvInject=true` 与 manager `EnableSyncGPUSharedResource=true`，核对目标版本的 feature gates。
4. 部署官方 hami-daemon，检查 libvgpu.so 在节点上的实际安装路径与版本。候选 Chart 0.1.0 / HAMi-core v2.6.0 需要按官方部署说明核对，不能只贴标签。
5. 只有库与运行时链路具备时，给 GPU 节点设置 `koordinator.sh/gpu-isolation-provider=HAMi-core`。
6. 构建 `workloads/Dockerfile.cuda`，以明确镜像名运行服务的 `-mode k8s-koord-hami -image <CUDA镜像>`，使用独立数据目录和专用 kubeconfig。
7. API 创建的 Pod 会使用 koord-scheduler，携带 HAMi-core provider 标签；申请 `koordinator.sh/gpu.shared: 1`、gpu-core、gpu-memory。绝对显存使用 MiB Quantity。
8. 逐项检查：Device CR → 分配注解 → 注入变量和挂载 → 应用进程 `/proc/PID/maps` → CUDA 调用 → 超限与邻居干扰。

`worker.py` 的真实模式要求 CUDA 可用且进程已映射 libvgpu；缺少任一项就失败。超限分配即使被拒绝，也需辨别 HAMi 限制与物理显存不足，不能仅根据一次 OOM 宣称隔离有效。

参考：
- https://koordinator.sh/docs/user-manuals/device-scheduling-gpu-share-with-hami
- https://koordinator.sh/docs/installation
- https://github.com/Project-HAMi/HAMi-core
- https://github.com/containerd/containerd/blob/main/docs/NRI.md
