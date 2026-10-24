# GPU 资源工作台

一个可运行、可修改、可排障的 Koordinator / HAMi-core 协议学习项目。目标是教会你解释一次任务从提交、配额预留、设备调度到停止回收的全过程。

当前本机：真实 Linux 虚拟机、Kubernetes、Koordinator 与 CPU 任务；两张 8 GiB GPU 及其显存约束为模拟。**没有运行真实 CUDA / HAMi-core 隔离，不能据此给出 GPU 性能结论。**

## 打开与学习

- 工作台：http://127.0.0.1:8787
- 逐课说明：http://127.0.0.1:8787/learn.html
- Prometheus：http://127.0.0.1:9090
- 在线 CPU 计算：http://127.0.0.1:18080/compute（需端口转发）

这些地址用于本机浏览器；手机上的 localhost 指向手机本身。

登录令牌保存在 `.runtime/credentials.json`，分别对应 alpha / beta。不要把该文件提交到 Git。网页只在当前浏览器会话保存令牌。

建议先做学习路径第 1–3 课：提交一个 30 秒、1024 MiB、25% 任务；查看团队额度、Pod 分配注解和日志；等任务结束确认额度归零。然后解释：**任务已通过团队额度检查，为什么仍可能 Pending？**

## 已提供的能力

- 团队令牌、事务配额、幂等提交、跨团队访问隔离。
- 持久化任务、事件与资源预留；单进程文件锁，后端身份绑定。
- 确定性 Kubernetes Job 名称、持续对账、未知状态保留额度。
- 取消时暂停 Job，等待控制器确认且 Pod 停止后释放；保留暂停对象防止迟到创建。
- 真实 DeviceShare 调度模拟设备，展示实际分配注解。
- 显存超限模拟、资源碎片、共享、健康变化、重启与故障实验。
- 内嵌网页、日志查询、团队指标、真实 CPU 在线服务及基准脚本。

`local-demo` 仅验证控制逻辑；`k8s-cpu` 验证 CPU Job；`k8s-sim` 为当前实验；`k8s-koord-hami` 保留真实 GPU 接入协议，未在本机验证。

## 本机日常启停

在项目目录运行：

```sh
scripts/start.sh k8s-sim
scripts/stop.sh
```

停止平台不会删除 Kubernetes 工作负载；重新启动后继续对账。暂停整个实验节点需先取消活动任务，然后运行：

```sh
. scripts/env.sh
colima -p gwb stop
```

恢复节点：`scripts/vm-start.sh`。如监控或在线端口转发因节点停止而退出，重新运行相应命令。

## 从源码重新部署

已具备可用 Go（>= 1.24、CGO 和 C 编译器）、Python 3、macOS 虚拟化能力：

```sh
scripts/install-tools.sh
scripts/vm-start.sh
scripts/install-cluster.sh
make test
make build
scripts/start.sh k8s-sim
python3 scripts/monitor-start.py
```

当前 Mac 分配 VM 4 核、6 GiB、30 GiB 数据盘。虚拟机文件存放在固定学习目录下的 `.vm`；短路径 `/tmp/gwb-vm` 指向它，以缩短 Unix socket 路径。不要删除运行中的虚拟机目录。

Koordinator 固定使用 1.7.0 + K3s 1.31.11。1.8.0 的发布版本依赖更高版本 Kubernetes，详见 [版本记录](docs/versions.md)。镜像使用官方 GHCR。`scripts/helm-post-render.sh` 在本地单节点关闭网络拓扑管理器，并将不可调度队列的最长等待设置为 10 秒；这个重试参数用于小型实验，不应直接外推到大型集群。

当前工作台以 macOS 原生进程运行，连接专用 kubeconfig；不会使用用户默认集群。`deploy/workbench.yaml` 是可选的集群内部署产物，需要先构建并导入镜像；当前原生部署使用同一份 Go 源码。

最近一次结果见 [本机验收记录](docs/2026-10-24-validation.md)。

## 验证与实验

```sh
make test
scripts/go-local.sh test -race ./...
scripts/go-local.sh vet ./...
scripts/preflight.sh
python3 scripts/e2e.py
python3 scripts/check-batch-health.py
```

端到端实验要求没有其他活动任务，只创建/取消自身任务，不删除历史记录。结果与完整任务日志写入 `.runtime/evidence/e2e-*.json`；`passed: false` 保留失败现场。测试包含幂等、团队隔离、成功运行、超限、碎片、同卡共享、取消与重启恢复。

设备故障注入：

```sh
. scripts/env.sh
python3 scripts/sim-devices.py --unhealthy 0
python3 scripts/sim-devices.py             # 恢复两卡健康
```

健康变化影响后续调度，不等于自动终止已有工作负载。模拟脚本拒绝操作没有 `workbench.local/lab=true` 标签的节点。

CPU 在线服务与基准：

```sh
. scripts/env.sh
kubectl apply -f deploy/online.yaml
kubectl -n gpu-workbench port-forward service/workbench-online 18080:8080
# 另开终端执行：
python3 scripts/benchmark.py --label baseline --samples 100 --concurrency 4
```

分别在无压力、LS 压力、BE 压力下用相同参数运行，保存原始样本。它们是 CPU / 主机调度实验，不是 GPU 利用率或吞吐测试；不要预设 BE 一定改善 P99。

## 文件导航

| 文件 | 阅读目的 |
|---|---|
| `internal/workbench/store.go` | 事务、幂等、额度、事件 |
| `internal/workbench/controller.go` | 观察优先、取消与恢复 |
| `internal/workbench/kube.go` | Job 模板、归属检查、实际分配与日志 |
| `internal/workbench/http.go` | 认证、API、指标 |
| `workloads/worker.py` | CPU 计算、模拟约束、真实 GPU 接入分支 |
| `scripts/e2e.py` | 完整实验与原始证据 |
| `docs/protocol.md` | 调度链与模拟边界 |
| `deploy/real-gpu.md` | 后续真实硬件验证步骤 |
| `docs/progress.md` | 实现裁定与验收记录 |

## 重要边界

团队配额只覆盖通过本平台提交的任务。它不是集群级租户隔离，也不是物理 GPU 用量。设备目录是最近读取到的注册信息，不代表实时健康探测。当前实现是单写入服务和本地 SQLite，没有高可用承诺。

终态 Job、事件和日志保留用于学习。取消请调用平台 API；不要删除 Pod/Job 来替代取消。准备清理时先停止平台、确认无活动任务并备份 `.runtime`。真实 GPU 模式需单独完成驱动、NRI、库映射、超限及邻居干扰验证。

## 可移植运行配置

Go 通过 PATH 查找，也可设置 `WORKBENCH_GO`。需要 Go 1.24 或更高版本、C 编译器及 Python 3。`GOCACHE` 和 `GOMODCACHE` 可指定外部缓存；默认使用忽略目录 `.cache`。

浏览器检查使用 `node scripts/visual-check.cjs`，先运行 `npm install --no-save --package-lock=false playwright` 和 `npx playwright install chromium`，或用 `PLAYWRIGHT_PATH` 指定已安装模块。`CHROME_PATH` 可指定浏览器。检查需要本地工作台、生成的本地测试凭据及 `.runtime/evidence` 目录；凭据和运行证据不会随源码发布。公网部署参考 `deploy/showmax/README.md`。
