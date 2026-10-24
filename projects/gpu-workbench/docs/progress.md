# 执行记录

计划：2026-10-23-plan.md
用户授权：已确认设计，并明确要求继续实现及本机部署。
Ruling: 用户已授权安装依赖及使用模拟 GPU；增加 k8s-sim 模式，部署本地 Linux VM、真实 Kubernetes 与 Koordinator。Device CR 为模拟设备，计算负载实际使用 CPU，HAMi-core 约束由显式模拟器解释；不得宣称真实 CUDA 隔离通过。
Ruling: 取消采用暂停 Job 并等待 Pod 终止，保留确定性名称的暂停对象防止迟到的创建重新启动；占用少量终态对象存储，提供清理说明。
Task 1: complete — Store 并发配额、幂等、身份隔离、持久化和终态释放测试已通过。
Task 2: in progress — Controller/Demo 的故障测试已观察到预期失败，正在实现。
裁定：新项目已有独立目录，无既有代码或共享分支，直接在该目录建立独立 Git 仓库。
Task 2: complete — Controller/Demo 和 Kubernetes 后端测试已通过，覆盖归属冲突、取消等待、孤儿 Pod、完成后分配信息保留。
Task 3: complete — API 测试通过，原生服务运行于 127.0.0.1:8787；增加数据库后端身份绑定与相对路径回归测试。
Ruling: Koordinator 1.8.0 官方发布源码依赖 K8s 1.35.2，DRA 锁定开启；改用 1.7.0 配合 K3s 1.31.11。成本是暂不覆盖 1.8 新功能，不影响本次 DeviceShare 与 HAMi-core 接入协议学习。
Ruling: 官方阿里云镜像仓库在主机及虚拟机均超时，使用官方 release workflow 中的 GHCR 镜像。
Ruling: Mac 系统代理干扰 Python urllib 的回环访问，所有本地探活和实验显式直连 localhost。
Task 4: in progress — 虚拟机、Kubernetes、Koordinator 已部署；正在运行端到端实验和监控、教学交付。
Ruling: 端到端实验发现 AssignedPodDelete 重试发生在设备缓存更新前，随后进入默认最长 5 分钟的不可调度队列；本地实验将受支持的 pod-max-in-unschedulable-pods-duration 设置为 10 秒，并重新验收。代价为更多重试，只用于小型实验环境，未修改 Koordinator 源码。
Ruling: 单节点无需集群网络拓扑管理器，通过 Helm post-renderer 关闭以消除缺少拓扑 CR 的重复日志；该配置不用于多节点拓扑实验。
Review: complete — 独立审查发现取消确认竞态、网页重试幂等键丢失、手工环境加载路径错误；三项均修复并完成只读复核。
Ruling: 取消必须等待控制器确认及无活动 Pod，不能仅依赖 spec.suspend；终态条件也可证明执行已停止。
Ruling: 当前阶段交付本机模拟部署，真实 CUDA/NRI 隔离及 LS/BE 性能改善结论保留为未验证；完整范围见 2026-10-24-validation.md。
Task 4: complete — 原生部署、9 项集群端到端、BE/设备健康实验、监控与页面操作已通过；学习页含 8 个逐步实验，等待学习者反馈检验掌握程度。
