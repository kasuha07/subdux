# Subdux

Subdux 是一个单二进制部署的订阅管理系统：后端使用 Go + Echo + GORM（SQLite），前端使用 React + Vite，构建后前端静态资源会被嵌入到后端可执行文件中。

## 功能概览

- 订阅管理：新增/编辑/删除、仪表盘汇总
- 多币种与汇率：汇率刷新与偏好货币
- 账户安全：密码登录、TOTP、Passkey、OIDC
- 通知系统：邮件 / Webhook / Push 渠道、策略、模板、日志
- 日历订阅：公开日历 feed + token 管理
- 导入导出：支持 Subdux 与 Wallos
- 管理后台：用户、系统配置、备份/恢复
- API Key：创建与权限范围控制

## 架构与运行方式

- 后端入口：`cmd/server/main.go`
- 前端构建产物：`web/dist/`
- 嵌入方式：`frontend.go` 中 `//go:embed all:web/dist`
- 路由：
  - `/api/*`：REST API
  - `/uploads/*`：上传资源
  - `/`：SPA 页面（含 fallback）

> 注意：由于使用 `go:embed`，在运行 `go run`/`go build` 之前必须先生成 `web/dist`。

## 环境要求

- Go **1.25+**
- Bun **1.x**
- （可选）tmux：用于 `make dev` 一键双窗口开发

## 快速开始（本地开发）

```bash
# 1) 先构建前端（生成 web/dist，供 go:embed 使用）
make frontend

# 2) 启动开发环境（tmux 中同时启动后端 + Vite）
make dev
```

如果你不使用 tmux，也可以手动启动：

```bash
# 终端 1（项目根目录）
go run ./cmd/server

# 终端 2
cd web
bun dev
```

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

## Docker

```bash
# 构建镜像
make docker

# 或使用 compose
docker compose up --build -d
```

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

## 项目结构

```text
subdux/
├── cmd/server/          # Go 入口
├── internal/            # 后端业务代码（api/service/model/pkg）
├── web/                 # React 前端
├── frontend.go          # 前端 embed 定义
├── Makefile             # 常用构建命令
└── docker-compose.yml
```

## 相关文档

- 前端子项目说明：[`web/README.md`](web/README.md)
- 协作约定：`AGENTS.md`（根目录及子目录）
