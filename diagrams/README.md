# 课程知识主图

每个目录保留 `candidate.json` 和可直接打开的 `diagram.html`。前两张概念图分别覆盖 HAMi 与 Koordinator；实践图用于联合证据链，运行时图用于显存预算。

安装 Archify 后，从仓库根目录运行（将工具路径替换为实际位置）：

```sh
node /path/to/archify/bin/archify.mjs finalize architecture diagrams/hami-overview/candidate.json .archify/architecture-evidence-reorder-20261026/diagram.html --quality showcase --json
```

其他图的模式见 candidate 中 `diagram_type`；输出位置以 `meta.output` 为准。仓库保留已生成页面；无需为浏览课程重新生成。

显存原理课新增 `hami-handoff` 配置交付图，更新 `hami-runtime` 为源码关联的时序图。两图分别固定 HAMi 与其引用的 HAMi-core 提交，重新生成时必须提供各自 `--repo-root`，详见显存机制实验的 source-lock.json。

`hami-pod-card`：第一课的 Pod 请求到启动配置交接图，固定 HAMi v2.7.0 源码；Kubernetes 与运行时内部过程折叠在箭头中。
