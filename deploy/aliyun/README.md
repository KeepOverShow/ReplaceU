# 阿里云部署

适用于有 SSH root 管理权限的 Debian/Ubuntu ECS。先安装 Python 3 与 Caddy（使用系统发行版软件包或 Caddy 官方安装说明），DNS 指向服务器并开放 HTTPS 443。课程只监听回环 18900。

```sh
DEPLOY_HOST=root@your-server COURSE_DOMAIN=learn.example.com bash deploy/aliyun/deploy.sh
```

首次部署会创建受限用户与 systemd 服务并启用自启动。将 `Caddyfile.example` 的域名变量替换为实际域名，合并到现有 Caddy 配置；先执行 `caddy validate --config /etc/caddy/Caddyfile`，再 reload Caddy。已有站点需保留原配置；脚本不会覆盖代理配置或停止原工作台。

原工作台入口预期位于回环 18787；其部署和 SSH 隧道见 `projects/gpu-workbench/deploy/showmax/`。课程可独立使用，工作台不可达只会显示状态提示。不要向公网开放无鉴权 Kubernetes API。

更新静态页面不重启服务。修改 server.py 或环境变量后，管理员需在维护窗口重启 hami-course 使其生效。当前已有服务保持运行；新域名需在 COURSE_ORIGINS 中配置允许来源。
