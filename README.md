# Subdux

Subdux 是一个面向自托管场景的订阅管理系统：后端使用 Go + Echo + GORM（SQLite），前端使用 React + Vite，构建后会把 `web/dist/` 嵌入到后端可执行文件中，最终以**单二进制 + 内嵌 SPA** 的方式部署。

它适合跟踪 SaaS、流媒体、域名、服务器、开发工具等周期性支出，并通过邮件、Webhook 与多种推送渠道提前提醒续费。

## 功能概览

### 订阅与账单
- 管理周期性与一次性账单，支持下次扣费日期、备注、分类、支付方式等字段
- 仪表盘汇总、筛选与月度成本展示
- 周期自动推进、到期提醒与日历视图
- 内置品牌图标选择，并支持上传自定义图标（可由管理员禁用）

### 多币种与汇率
- 记录多币种订阅金额
- 偏好货币换算与汇总展示
- 汇率数据刷新与后台定时更新

### 认证与安全
- 账号密码登录、注册、忘记密码、重置密码
- TOTP 双因素认证与一次性恢复码
- Passkey / WebAuthn 无密码登录
- OIDC（OpenID Connect）单点登录接入
- API Key（读 / 写权限范围）
- 安全响应头、认证限流、认证请求体大小限制、敏感查询参数日志脱敏
- 系统设置与通知渠道敏感字段支持加密存储

### 通知系统
- 续费前 N 天提醒与到期日提醒策略
- 通知模板、预览、测试发送与日志
- 支持多种通知渠道：
  - 邮件：SMTP、Resend
  - 机器人 / Webhook：Telegram、Webhook、Feishu、WeCom、DingTalk、NapCat
  - 推送：Gotify、ntfy、Bark、ServerChan3、PushDeer、pushplus、Pushover

### 管理与运维
- 管理后台：用户管理、系统设置、统计、备份 / 恢复
- 注册开关、注册邮箱验证码、邮箱域名白名单
- SMTP 与 OIDC 可在后台配置
- 汇率来源与站点信息可在后台维护

### 导入、导出与集成
- 支持 Subdux 原生 JSON 导出 / 导入
- 支持 Wallos JSON 导入
- 提供带 token 的只读日历订阅 feed
- 提供可供外部程序访问的 API Key
- 前端内置多语言：`en`、`zh-CN`、`ja`

## 架构与运行方式

- 后端入口：`cmd/server/main.go`
- 前端构建产物：`web/dist/`
- 嵌入方式：`frontend.go` 中 `//go:embed all:web/dist`
- 路由分工：
  - `/api/*`：REST API
  - `/uploads/*`：上传资源
  - `/`：SPA 页面（含 fallback）
- 数据目录：默认 `./data`，可用 `DATA_PATH` 覆盖
- 后台任务：启动后会自动处理待发送通知，并定时刷新汇率

> 注意：由于使用 `go:embed`，在运行 `go run` / `go build` 之前必须先生成 `web/dist`。

## 技术栈

- **Backend**：Go 1.25、Echo v4、GORM、SQLite
- **Frontend**：React 19、Vite 7、TypeScript 5、Tailwind CSS v4、Shadcn/UI
- **Auth / Security**：JWT、TOTP、WebAuthn / Passkey、OIDC
- **Deployment**：单二进制部署、Docker 多阶段构建、distroless 运行镜像

## 环境要求

- Go **1.25+**
- Bun **1.x**
- （可选）tmux：用于 `make dev` 一键双窗口开发

## 快速开始（本地开发）

### 方式一：使用 Makefile

```bash
# 1) 先构建前端（生成 web/dist，供 go:embed 使用）
make frontend

# 2) 启动开发环境（tmux 中同时启动后端 + Vite）
make dev
```

> `make dev` 依赖 `tmux`。如果你不使用 tmux，请使用下面的手动方式。

### 方式二：手动启动

```bash
# 终端 1（项目根目录）
make frontend
go run ./cmd/server

# 终端 2
cd web
bun dev
```

默认情况下：
- 后端运行在 `http://localhost:8080`
- 前端开发服务器运行在 `http://localhost:5173`
- Vite 会把 `/api` 请求代理到后端

## 构建与测试

```bash
# 构建前后端并输出单文件二进制 ./subdux
make build

# 后端测试
go test ./...

# 前端检查
cd web
bun run lint
bun run build
```

## Docker / Compose

### 构建镜像

```bash
make docker
```

### 使用 Compose 启动

```bash
docker compose up --build -d
```

默认 Compose 配置会：
- 暴露 `8080` 端口
- 将数据目录挂载到命名卷 `subdux-data`
- 通过 `DATA_PATH=/data` 把 SQLite、上传文件和密钥文件保存在容器卷中

## GitHub Packages / GHCR

Subdux 会在推送语义化版本标签（例如 `v0.8.0`）时，通过 GitHub Actions 自动发布容器镜像到 GitHub Container Registry（GHCR）。

- 包地址：`ghcr.io/kasuha07/subdux`
- 发布触发器：`.github/workflows/docker-publish.yml` 中的 `push.tags: ["v*"]`
- 生成的主要镜像标签：`0.8.0`、`0.8`（由 Git 标签 `v0.8.0` 映射而来）

### 拉取镜像

```bash
docker pull ghcr.io/kasuha07/subdux:0.8.0
```

如果包保持私有，请先登录 GHCR：

```bash
echo "<YOUR_GITHUB_TOKEN>" | docker login ghcr.io -u <YOUR_GITHUB_USERNAME> --password-stdin
```

### 运行 GHCR 镜像

```bash
docker run -d \
  --name subdux \
  -p 8080:8080 \
  -e DATA_PATH=/data \
  -v subdux-data:/data \
  ghcr.io/kasuha07/subdux:0.8.0
```

### 维护者发布步骤

```bash
git tag v0.8.0
git push origin v0.8.0
```

发布出的镜像会写入 OCI 元数据（仓库来源、文档链接、许可证、版本和提交信息），便于在 GHCR 页面中追溯来源。

首次发布后，GitHub Package 可能默认是私有可见；如果需要匿名 `docker pull`，请到 GitHub 仓库的 **Packages / Package settings** 中将其切换为 **Public**。

## 关键环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | 服务监听端口 |
| `DATA_PATH` | `data` | 数据目录（SQLite、上传文件、密钥文件） |
| `JWT_SECRET` | 空（自动生成并写入 DB） | JWT 密钥，建议生产环境显式配置（至少 32 字符） |
| `ACCESS_TOKEN_TTL_MINUTES` | `15` | Access Token 过期分钟数（最小 1） |
| `REFRESH_TOKEN_TTL_HOURS` | `720` | Refresh Token 过期小时数（最小 1） |
| `SETTINGS_ENCRYPTION_KEY` | 空 | 系统设置加密密钥；未设置时回退 `JWT_SECRET`，再回退本地生成密钥文件 |
| `CORS_ALLOW_ORIGINS` | 空 | 逗号分隔 origins；设置后覆盖默认 CORS 来源逻辑 |
| `TZ` | 系统时区 | 时区（IANA 名称，如 `UTC`、`Asia/Shanghai`） |

### 生产环境建议

- 为 `JWT_SECRET` 与 `SETTINGS_ENCRYPTION_KEY` 提供稳定且高强度的值
- 通过反向代理或公网域名配置正确的 `site_url` / CORS 来源
- 将 `DATA_PATH` 挂载到持久化存储
- 若启用注册邮箱验证码、密码重置或邮件通知，请先配置 SMTP
- 若启用 OIDC，请确保回调地址与身份提供方配置完全一致

## 项目结构

```text
subdux/
├── cmd/server/          # Go 服务入口
├── internal/            # 后端业务代码（api / service / model / pkg）
├── web/                 # React 前端
├── frontend.go          # //go:embed 前端产物
├── Makefile             # 常用构建命令
├── Dockerfile           # Bun → Go → distroless 多阶段镜像
└── docker-compose.yml
```

## 常用命令

### 后端

```bash
go run ./cmd/server
go test ./...
go build -o subdux ./cmd/server
```

### 前端

```bash
cd web
bun install
bun dev
bun run build
bun run lint
```

## 相关文档

- 前端子项目说明：[`web/README.md`](web/README.md)
- 协作约定：`AGENTS.md`（根目录及子目录）
