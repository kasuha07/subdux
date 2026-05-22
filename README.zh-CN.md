# Subdux

[![CI](https://github.com/kasuha07/subdux/actions/workflows/ci.yml/badge.svg)](https://github.com/kasuha07/subdux/actions/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fkasuha07%2Fsubdux-2ea44f?logo=docker)](https://github.com/kasuha07/subdux/pkgs/container/subdux)
[![License: GPL-3.0-or-later](https://img.shields.io/badge/License-GPL--3.0--or--later-blue.svg)](LICENSE)

**语言：** [English](README.md) | 简体中文

**Subdux** 是一个面向自部署 / 自托管场景的订阅管理系统，用来跟踪周期性账单、续费日期和提醒通知。
它将 Go 后端与 React 前端打包为一个带内嵌 SPA 的**单一可部署二进制**，同时也支持 Docker 容器部署，适合个人服务器、家庭实验室和小型生产环境。

你可以用它来管理 SaaS、域名、流媒体、云服务器、开发工具订阅，以及其他任何周期性支出，并在续费前收到提醒。

## 亮点

- **为自托管而设计** —— 可直接运行单二进制，也可运行容器。
- **专注订阅场景** —— 支持周期性订阅、仪表盘汇总、分类、支付方式、图标与日历视图。
- **支持多币种** —— 可记录不同币种的订阅，并按偏好币种汇总展示。
- **内置提醒系统** —— 提醒策略、模板、预览、测试发送、日志全部具备。
- **现代认证能力** —— 账号密码、密码重置、TOTP 双因素、Passkey / WebAuthn、OIDC、带作用域的 API Key。
- **具备管理后台** —— 用户管理、注册控制、SMTP / OIDC 设置、汇率管理、统计、备份恢复。
- **数据可迁移** —— 支持 Subdux 原生 JSON 导入导出、Wallos 导入与日历订阅链接。

## 功能概览

| 模块 | 能力 |
| --- | --- |
| 订阅追踪 | 周期性订阅、下次扣费日期、备注、分类、支付方式、图标、仪表盘汇总 |
| 通知系统 | 续费提醒、按天提醒策略、模板、预览、测试发送、投递日志 |
| 通知渠道 | SMTP、Resend、Telegram、Webhook、Gotify、ntfy、Bark、ServerChan3、PushDeer、pushplus、Pushover、Feishu、WeCom、DingTalk、NapCat |
| 认证 | 邮箱 / 密码、忘记密码 / 重置密码、TOTP + 恢复码、Passkey / WebAuthn、OIDC、API Key |
| 管理功能 | 用户管理、注册控制、邮箱域名白名单、SMTP 设置、OIDC 设置、统计、备份 / 恢复 |
| 导入 / 导出 | Subdux 原生导入导出、Wallos 导入、日历 token、API 访问 |
| 多语言 | English、简体中文（`zh-CN`）、日本語（`ja`） |

## 快速开始

### 方式一：运行已发布的容器镜像

将 `<version>` 替换为实际版本号，例如 `0.8.1`。

```bash
docker run -d \
  --name subdux \
  -p 8080:8080 \
  -e DATA_PATH=/data \
  -e JWT_SECRET=replace-with-a-long-random-string \
  -v subdux-data:/data \
  ghcr.io/kasuha07/subdux:<version>
```

启动后访问 <http://localhost:8080>。

> 全新实例中，**第一个注册用户会自动成为管理员**。

### 方式二：使用仓库自带 Docker Compose

仓库内置的 `docker-compose.yml` 会在本地构建镜像。

```bash
docker compose up --build -d
```

默认会监听 `8080` 端口，并将持久化数据保存到 `subdux-data` 卷中。

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

### 生产环境建议

- 将 `DATA_PATH` 挂载到持久化存储。
- 在生产环境中设置稳定的 `JWT_SECRET` 和 `SETTINGS_ENCRYPTION_KEY`。
- 对外部署时配置 `CORS_ALLOW_ORIGINS` 和 / 或系统内的 `site_url`。
- 启用邮箱验证、密码重置或邮件通知前，先完成 SMTP 配置。
- 如果启用 OIDC，确保 Subdux 中的回调地址与身份提供方配置完全一致。
- Passkey 和 OIDC 通常要求正确的公网 URL 与 HTTPS 配置。

## 架构说明

Subdux 虽然是 monorepo，但最终以单应用方式部署：

- **后端：** Go 1.25 + Echo + GORM + SQLite
- **前端：** React 19 + Vite + TypeScript + Tailwind CSS v4 + Shadcn/UI
- **部署模型：** 前端先构建到 `web/dist`，再通过 `go:embed` 嵌入 Go 二进制

运行时路由：

- `/` —— SPA 前端
- `/api/*` —— REST API
- `/uploads/*` —— 上传资源
- `/api/calendar/feed` —— 带 token 的只读日历订阅地址

服务启动后会自动运行以下后台任务：

- 汇率刷新
- 待发送通知处理

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
└── docker-compose.yml
```

## 开发

### 环境要求

- Go **1.25+**
- Bun **1.x**
- 可选：`tmux`（用于 `make dev`）

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
go test ./...

cd web
bun run lint
bun run build
```

## 发布与分发

- CI 会在推送到 `main` 或发起 PR 时运行。
- `v0.8.1` 这类版本标签会自动发布多架构 GHCR 镜像。
- 容器镜像地址：`ghcr.io/kasuha07/subdux`
- Releases：<https://github.com/kasuha07/subdux/releases>

## 贡献

欢迎提交 issue 和 pull request。

提交 PR 前，建议先运行：

```bash
go test ./...
cd web && bun run lint && bun run build
```

如果你修改的是 UI 或前端交互，请尽量遵循 `web/src/features/` 下的按业务域组织结构，并避免直接修改 `web/src/components/ui/` 中的生成文件。

## 许可证

Subdux 使用 **GPL-3.0-or-later** 许可证，详见 [`LICENSE`](LICENSE)。
