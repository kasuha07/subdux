# Subdux

[![CI](https://github.com/kasuha07/subdux/actions/workflows/ci.yml/badge.svg)](https://github.com/kasuha07/subdux/actions/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fkasuha07%2Fsubdux-2ea44f?logo=docker)](https://github.com/kasuha07/subdux/pkgs/container/subdux)
[![License: GPL-3.0-or-later](https://img.shields.io/badge/License-GPL--3.0--or--later-blue.svg)](../LICENSE)

<p align="center">
  <img src="../web/public/subdux-logo.svg" alt="Subdux" width="320">
</p>

<p align="center">
  <strong>下一笔续费，早知道，早决定。</strong>
</p>

<p align="center">
  自托管的周期支出决策中心——看清订阅、掌握续费、准时提醒，记录每一次取舍。
</p>
<p align="center">
  <a href="#快速开始"><strong>立即部署</strong></a> ·
  <a href="#自动化与-mcp">连接 AI Agent</a> ·
  <a href="#subdux-能做什么">了解产品</a> ·
  <a href="../README.md">English</a>
</p>

---

Subdux 把散落在各处的订阅周期扣费整理成一条清晰、可行动的时间线：下一笔什么时候扣、跨币种到底花了多少、应该续费还是取消，都能在账单发生前看见并做出决定。

单二进制交付，docker 一键部署。订阅记录、通知配置都保留在你自己的基础设施中。

## 截图预览

<table>
  <tr>
    <td width="33.33%" align="center">
      <img src="screenshots/dashboard.png" alt="Subdux 仪表盘">
      <br>
      <strong>仪表盘</strong>
    </td>
    <td width="33.33%" align="center">
      <img src="screenshots/action_center.png" alt="Subdux 行动中心">
      <br>
      <strong>行动中心</strong>
    </td>
    <td width="33.33%" align="center">
      <img src="screenshots/reports.png" alt="Subdux 报表分析">
      <br>
      <strong>报表分析</strong>
    </td>
  </tr>
  <tr>
    <td width="33.33%" align="center">
      <img src="screenshots/notification_settings.png" alt="Subdux 通知设置">
      <br>
      <strong>通知设置</strong>
    </td>
    <td width="33.33%" align="center">
      <img src="screenshots/calendar_view.png" alt="Subdux 日历视图">
      <br>
      <strong>日历视图</strong>
    </td>
    <td width="33.33%" align="center">
      <img src="screenshots/subscription_detail.png" alt="Subdux 订阅详情">
      <br>
      <strong>订阅详情</strong>
    </td>
  </tr>
</table>

## 为什么选择 Subdux

- **轻量部署。** 前端、后端与后台任务集成在单个 Go 二进制中，使用 SQLite 持久化，无需额外部署数据库或缓存服务。
- **围绕续费决策，而不只是记录订阅。** 行动中心集中管理待处理事项，续费与修改历史保留每次决策后的变化。
- **可配置的完善通知体系。** 提醒策略、免打扰时段、模板、预览、测试发送和投递日志，与 15 种通知渠道形成完整闭环。
- **数据来去自由，不被平台绑定。** 支持完整导入导出、Wallos 迁移、日历订阅与备份恢复，方便迁移和长期保存。
- **将订阅管理接入 AI Agent 工作流。** 通过 MCP 查询、创建、更新和续费订阅，并为写操作提供可靠的重试保障。

## 快速开始

### Docker CLI

下载 [`.env.example`](../.env.example) 并重命名为 `subdux.env`，再为 `JWT_SECRET` 和 `SETTINGS_ENCRYPTION_KEY` 分别生成稳定的随机密钥：

```bash
curl -fsSL \
  "https://raw.githubusercontent.com/kasuha07/subdux/main/.env.example" \
  -o subdux.env
chmod 600 subdux.env
sed -i \
  -e "s|^JWT_SECRET=.*$|JWT_SECRET=$(openssl rand -hex 32)|" \
  -e "s|^SETTINGS_ENCRYPTION_KEY=.*$|SETTINGS_ENCRYPTION_KEY=$(openssl rand -hex 32)|" \
  subdux.env
mkdir -p data

docker run -d \
  --name subdux \
  --restart unless-stopped \
  -p 8080:8080 \
  --env-file ./subdux.env \
  -e DATA_PATH=/data \
  -v "$(pwd)/data:/data" \
  ghcr.io/kasuha07/subdux:stable
```

容器启动后访问 <http://localhost:8080>。

全新实例会在启动时自动创建初始管理员，并且只在首次初始化时打印随机密码：

```bash
docker logs subdux
```

默认初始管理员：

- 用户名：`admin`
- 邮箱：`admin@subdux.local`

如果想自行指定初始凭据，请在首次启动前修改 `subdux.env` 中的 `SUBDUX_INITIAL_ADMIN_USERNAME`、`SUBDUX_INITIAL_ADMIN_EMAIL` 和 `SUBDUX_INITIAL_ADMIN_PASSWORD`。默认关闭公开注册，登录管理员后可在管理设置中开启。

### Docker Compose

下载 Compose 与环境变量示例，为 JWT 和设置加密分别生成稳定密钥，然后启动服务：

```bash
curl -fsSLO \
  "https://raw.githubusercontent.com/kasuha07/subdux/main/docker-compose.yml"
curl -fsSL \
  "https://raw.githubusercontent.com/kasuha07/subdux/main/.env.example" \
  -o .env
chmod 600 .env
sed -i \
  -e "s|^JWT_SECRET=.*$|JWT_SECRET=$(openssl rand -hex 32)|" \
  -e "s|^SETTINGS_ENCRYPTION_KEY=.*$|SETTINGS_ENCRYPTION_KEY=$(openssl rand -hex 32)|" \
  .env
mkdir -p data

docker compose up -d
```

容器启动后，访问 <http://localhost:8080>。首次启动时，通过日志获取只显示一次的初始管理员密码：

```bash
docker compose ps
docker compose logs subdux
```

请妥善保管 `.env`，并备份 `data/` 目录。升级只需拉取镜像并重建容器：

```bash
docker compose pull
docker compose up -d
```

## 自动化与 MCP

在 Subdux 设置中创建 API Key，然后连接 MCP 客户端：

```json
{
  "mcpServers": {
    "subdux": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": {
        "X-API-Key": "sdx_xxx...",
        "Content-Type": "application/json",
        "Accept": "application/json"
      }
    }
  }
}
```

端点保持无状态：每个 `POST /mcp` 请求都会独立认证。写操作工具必须提供 `idempotency_key`，客户端在超时后可以安全重试而不会重复执行；同一个 Key 携带不同参数会被拒绝，且 Key 按用户隔离。

Subdux 为 `initialize`、`ping`、`tools/list` 和 `tools/call` 返回 JSON-RPC 响应，不维护 MCP 传输会话，也不提供 SSE 或服务端主动流式推送。

## Subdux 能做什么

**预览周期支出**

添加订阅时，可以记录金额、币种、计费周期、续费方式、下次扣费日、分类、支付方式、备注和自定义图标。无论是按周、按月、按年，还是自定义周期的 SaaS、域名、云服务和会员，都可以用同一套模型管理，而不必维护零散的表格和日历提醒。

仪表盘会按偏好币种汇总月度与年度成本，并展示近期扣费；报表进一步拆分分类、支付方式和续费方式，给出未来 12 个月的扣费预测、近期涨价和年度成本变化。日历视图则把每次预计扣费放回时间线上，让“花了多少”和“什么时候要处理”能够相互对应。

**围绕续费决策，而不只是记录订阅**

行动中心不只展示“即将到期”，还会集中呈现即将自动扣费、等待手动续费、即将终止、价格上涨、通知失败和缺少扣费日等需要处理的事项，并按紧急程度和时间窗口组织。你可以直接标记续费、继续保留、设置到期终止，或暂时忽略七天，而不需要在多个页面之间寻找对应订阅。

每次创建、修改和手动续费都会留下变更记录，订阅详情也会展示后续扣费安排。对于自动续费和到期终止的订阅，后台任务会持续推进生命周期，使仪表盘、报表和行动中心反映当前状态，而不是停留在最初录入的数据上。

**把通知送到你真正使用的地方**

通知策略可以控制提前提醒天数、到期日提醒和手动续费提醒；单个订阅也可以覆盖默认设置。免打扰时段会在真正投递前再次检查，避免已经进入队列的通知绕过最新策略。

你可以为所有渠道设置默认模板，也可以为特定渠道定制内容，并在启用前预览或测试发送。通知会先进入持久化队列，投递失败后按策略重试，最终结果写入日志，便于判断提醒是否真正送达。渠道覆盖 SMTP、Resend、Telegram、Webhook、Gotify、ntfy、Bark、Server酱3、PushDeer、pushplus、Pushover、飞书、企业微信、钉钉和 NapCat。

**多用户能力**

每个用户可以管理自己的订阅、通知和访问凭据，并通过密码、TOTP 与恢复码、Passkey / WebAuthn 或 OIDC 登录。公开注册默认关闭；需要多人使用时，管理员可以控制注册开关、允许的邮箱域名、用户角色和账号状态。

账号凭据、用户管理、系统设置和备份恢复等敏感操作受到人类会话与操作级二次认证保护。供程序使用的 API Key 则拥有独立作用域，不会被当作人类登录会话使用。管理员还可以查看审计记录、后台任务状态，并集中配置 SMTP、OIDC、汇率来源和备份策略。

**数据随时可以带走**

Subdux 原生 JSON 可用于迁移订阅、分类、支付方式、币种偏好与通知配置，敏感通知密钥默认会被脱敏；确实需要完整迁移时，也可以在重新验证身份后导出密钥。导入前会展示变更预览，便于确认新增、更新和跳过的内容。已有 Wallos 用户也可以先预览迁移结果，再导入订阅、分类、币种和支付方式。

日历 Feed 通过独立 Token 提供只读订阅，可以接入常用日历客户端。管理员还可以创建或定时生成实例备份，选择是否包含上传资源、启用加密并设置保留数量。结合 REST API 和 MCP，数据既能完整迁移，也能继续被其他工具安全使用。

## 配置

### 关键环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | HTTP 监听端口 |
| `DATA_PATH` | `data` | SQLite 数据库、上传资源和本地生成密钥的存储目录 |
| `JWT_SECRET` | 未设置时首次启动自动生成 | 生产环境建议显式配置，且长度至少 32 字符 |
| `SETTINGS_ENCRYPTION_KEY` | 依次回退到 `JWT_SECRET` 与本地生成密钥文件 | 用于加密系统设置和通知渠道中的敏感信息 |
| `ACCESS_TOKEN_TTL_MINUTES` | `15` | Access Token 有效期 |
| `REFRESH_TOKEN_TTL_HOURS` | `720` | Refresh Token 有效期 |
| `CORS_ALLOW_ORIGINS` | 未设置 | 逗号分隔的允许来源列表 |
| `TZ` | 系统时区 | IANA 时区名称，例如 `UTC`、`Asia/Shanghai` |
| `LOG_LEVEL` | `info` | 最低日志级别：`debug`、`info`、`warn` 或 `error` |
| `LOG_FORMAT` | `auto` | 日志输出格式：`json`、`text` 或 `auto`（连接终端时输出可读文本，否则输出 JSON） |
| `SUBDUX_INITIAL_ADMIN_USERNAME` | `admin` | 初始管理员用户名，仅在用户表为空时使用 |
| `SUBDUX_INITIAL_ADMIN_EMAIL` | `admin@subdux.local` | 初始管理员邮箱，仅在用户表为空时使用 |
| `SUBDUX_INITIAL_ADMIN_PASSWORD` | 随机生成并打印一次 | 初始管理员密码；未设置时会生成随机密码并在首次启动时打印 |

### 生产环境建议

- 将 `DATA_PATH` 挂载到持久化存储。
- 在生产环境中设置稳定的 `JWT_SECRET` 和 `SETTINGS_ENCRYPTION_KEY`。
- 对外部署时配置 `CORS_ALLOW_ORIGINS` 和/或系统内的 `site_url`。
- 启用邮箱验证、密码重置或邮件通知前，先完成 SMTP 配置。
- 如果启用 OIDC，确保 Subdux 中的回调地址与身份提供方配置完全一致。
- Passkey 和 OIDC 通常要求正确的公网 URL 与 HTTPS 配置。
- 如果启用应用内的系统代理，请在代理侧配置出站 ACL。对于用户可配置的出站请求，Subdux 会在代理前应用主机名策略；但代理 DNS 解析和私网地址阻断由代理负责。管理员配置的出站目标视为管理员策略并被信任。

## 架构说明

- **后端：** Go 1.26.4 + Echo + GORM + SQLite
- **前端：** React 19 + Vite + TypeScript + Tailwind CSS v4 + Shadcn/UI
- **部署模型：** 前端先构建到 `web/dist`，再通过 `go:embed` 嵌入 Go 二进制

## 项目结构

```text
subdux/
├── cmd/server/          # Go 服务入口
├── internal/            # 后端代码
│   ├── api/             # Echo 路由与处理器
│   ├── model/           # GORM 模型
│   ├── pkg/             # 基础设施与共享工具
│   └── service/         # 业务逻辑
├── web/                 # React 前端
├── frontend.go          # web/dist 的 go:embed 入口
├── Makefile             # 常用构建命令
├── Dockerfile           # 多阶段容器构建
├── docker-compose.yml   # Compose 部署示例
└── .env.example         # 环境变量示例
```

## 开发

### 环境要求

- Go **1.26.4+**
- Bun **1.x**
- 可选：使用 `make dev` 时需要 tmux

### 本地开发

由于前端会被嵌入到后端二进制，所以直接运行 Go 服务前需要先生成 `web/dist`。

```bash
# 先构建前端产物，供 go:embed 使用
make frontend

# 启动后端
go run ./cmd/server
```

前端开发可在第二个终端中运行：

```bash
cd web
bun dev
```

默认开发地址：

- 后端：<http://localhost:8080>
- 前端开发服务器：<http://localhost:5173>
- Vite 会将 `/api` 请求代理到后端

### Make 命令

```bash
make frontend   # bun install + 前端生产构建
make build      # 前端构建 + Go 二进制构建
make dev        # 用 tmux 同时启动后端和 Vite
make docker     # 本地构建 Docker 镜像
make clean      # 删除本地二进制
```

### 检查命令

```bash
make check              # 格式、前端 lint/build、Go vet/test
cd web && bun run test  # 前端测试
```

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

Subdux 使用 **GPL-3.0-or-later** 许可证，详见 [`LICENSE`](../LICENSE)。
