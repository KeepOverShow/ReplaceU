# 依赖与验证矩阵

- 主机：Intel Mac，macOS 15.5，16 GiB RAM；Intel Iris Plus，不具备 NVIDIA CUDA。
- 本地虚拟机：Colima 0.9.1、Lima 1.2.1，VZ，4 vCPU / 6 GiB RAM / 30 GiB 数据盘。
- Kubernetes：K3s v1.31.11+k3s1；工作台 client-go v0.31.0。
- Go：当前主机 1.27.2；go.mod 声明最低 1.24。
- SQLite 驱动：github.com/mattn/go-sqlite3 v1.14.24，CGO。
- 工作负载镜像：python:3.12.11-slim-bookworm，CPU 真实执行。
- Koordinator：1.7.0，官方 GHCR 镜像；三个组件已就绪。1.8.0 依赖 Kubernetes 1.35.2，默认 DRA 已锁定开启，不能用于本实验的 1.31 API；已回退至支持 HAMi-core 接入的 1.7.0。
- 真实 HAMi-core 候选：v2.6.0，hami-daemon Chart 0.1.0；本机未执行真实 CUDA 隔离测试。

不要将“配置文件提供”视为“真实硬件验证通过”。当前模式中没有使用独立 HAMi Scheduler。
