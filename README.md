# ReplaceU

系统学习云计算：先学会原理，再用服务、实验和源码核对理解。

## 三部分课程

| 部分 | 内容 | 课程源码 |
|---|---|---|
| 第一部分 | HAMi 组件、调度链路、HAMi-core 显存预算 | [HAMi](courses/hami/) |
| 第二部分 | Koordinator 组件、观测、调度与节点治理 | [Koordinator](courses/koordinator/) |
| 第三部分 | Koordinator + HAMi-core 联合实践、资源碎片、证据链 | [实践](courses/practice/) |

当前提供 5 节入门课。每课配 Archify 交互主图、讲解与复述任务；后续主题逐步增加，不把目录当作已掌握记录。

- [在线目录](https://showmax.cloud/course/catalog.html)
- [课程站点与模拟 API](apps/course-site/)
- [阿里云部署](deploy/aliyun/)
- [完整实践项目](projects/gpu-workbench/)
- [HAMi 显存机制模型、源码校验与 CUDA 探针](projects/hami-memory-lab/)
- [Archify 图源与页面](diagrams/)

## 本地启动

```sh
python3 apps/course-site/build.py
python3 apps/course-site/server.py
```

服务监听 127.0.0.1:18900，网页公开路径为 `/course/`，需通过反向代理剥离该前缀；本地预览使用 `python3 apps/course-site/preview.py`，再打开 http://127.0.0.1:18901/course/。

```sh
cd apps/course-site
python3 -m unittest -v
```

实践项目的 Go、Kubernetes 和集群部署步骤见其 README。普通 CPU 环境可以运行模拟流程，真实 NVIDIA 场景还需硬件、驱动、运行库和对应版本集成验证。课程 API 是有限的算术模型，不调用 CUDA，不写入集群。

图源使用 [Archify](https://github.com/tt-a1i/archify) 生成。已保留交互 HTML，不要求读者安装该工具。官方技术依据链接随课提供；具体行为以部署版本为准。
