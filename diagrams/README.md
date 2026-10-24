# 课程知识主图

每个目录保留 `candidate.json` 和可直接打开的 `diagram.html`。前两张概念图分别覆盖 HAMi 与 Koordinator；实践图用于联合证据链，运行时图用于显存预算。

安装 Archify 后，从仓库根目录运行（将工具路径替换为实际位置）：

```sh
node /path/to/archify/bin/archify.mjs finalize architecture diagrams/hami-overview/candidate.json .archify/architecture-hami-overview-20261024/diagram.html --quality showcase --json
```

其他图的模式见 candidate 中 `diagram_type`；输出位置以 `meta.output` 为准。仓库保留已生成页面；无需为浏览课程重新生成。
