# HTTPS 与反向隧道部署示例

链路：浏览器 HTTPS → 自有服务器的 Caddy → 服务器回环 18787 → SSH 反向隧道 → 本机回环 8787 工作台。

这是部署模板，不代表已经部署或验证的公网服务。设置 `WORKBENCH_DOMAIN` 为自己的域名后加载 Caddyfile。`/course/` 转发至回环 18900，其余路径转发至 18787。配置使用 TLS-ALPN 证书校验；确保 DNS 与 443 端口满足要求。

按 sshd-tunnel.conf 创建专用受限账户（示例名称 gwb-tunnel，可自行替换），部署公钥并通过可信渠道核验服务器主机指纹。密钥、known_hosts 和日志不得提交。

```sh
export WORKBENCH_SSH_TARGET='gwb-tunnel@your-server.example'
export WORKBENCH_SSH_KEY='/path/to/private-key'
export WORKBENCH_KNOWN_HOSTS='/path/to/verified-known-hosts'
# 可选，默认仅转发服务器回环 18787 至本机回环 8787
export WORKBENCH_SSH_FORWARD='127.0.0.1:18787:127.0.0.1:8787'
python3 scripts/showmax-tunnel.py
```

脚本在当前会话前台运行，断线 30 秒后重试，Ctrl-C 停止。PID 和文件锁写入忽略目录 `.runtime/showmax`。不提供开机自启；本机休眠、断网或工作台停止时入口不可用。工作台仍要求团队令牌。

Caddy 文档：https://caddyserver.com/docs/caddyfile/directives/tls
