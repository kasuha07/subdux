# Subdux Frontend

Subdux 前端基于 React 19 + Vite + TypeScript，构建产物会打包为 SPA，并在主项目构建阶段嵌入后端二进制中。

> 完整项目说明（后端、部署、Docker、环境变量）请查看根目录 [`README.md`](../README.md)。

## 前端能力概览

### 页面与业务模块
- 认证页面：登录、注册、忘记密码、重置密码
- 仪表盘：订阅列表、汇总卡片、筛选工具栏
- 设置中心：账户信息、TOTP、Passkey、OIDC、API Key、通知、分类、支付方式
- 日历页面：订阅日历视图与 calendar token 管理
- 管理后台：用户、系统设置、SMTP、OIDC、统计、备份 / 恢复

### 交互与基础设施
- `App.tsx` 中统一定义 `ProtectedRoute`、`PublicRoute`、`AdminRoute`
- `src/lib/api.ts` 统一处理 API 请求、Bearer Token、401 自动清理会话并跳转登录页
- i18next + 浏览器语言检测，内置 `en`、`zh-CN`、`ja`
- Shadcn/UI + Tailwind CSS v4 + Sonner 提供基础 UI、主题变量与提示反馈
- 品牌图标系统与上传图标能力共同支撑订阅 / 支付方式展示

## 技术栈

- React 19
- Vite 7
- TypeScript 5
- React Router 7
- Tailwind CSS v4
- Shadcn/UI
- i18next / react-i18next
- Lucide React + Sonner

## 环境要求

- Bun **1.x**
- Node 兼容环境（由 Bun 负责脚本执行）

## 本地开发

```bash
cd web
bun install
bun dev
```

默认行为：
- 开发地址：`http://localhost:5173`
- `/api` 请求代理到 `http://localhost:8080`
- 如需前后端联调，请在根目录另外启动 Go 服务（通常先执行 `make frontend`，再执行 `go run ./cmd/server`）

## 构建与检查

```bash
cd web
bun run lint
bun run build
bun run preview
```

- `bun run lint`：运行 ESLint
- `bun run build`：先执行 `tsc -b`，再执行 Vite 生产构建
- `bun run preview`：预览生产构建产物

## 目录结构

```text
web/
├── src/
│   ├── App.tsx               # 路由入口与访问守卫
│   ├── features/             # 按业务域组织页面与组件
│   ├── components/ui/        # Shadcn 生成组件（不要直接修改）
│   ├── lib/                  # API 封装、工具函数、品牌图标等
│   ├── i18n/                 # 多语言资源与初始化
│   └── types/                # 与后端 JSON 结构对齐的类型定义
├── package.json              # Bun 脚本与依赖
├── vite.config.ts            # React + Tailwind 插件、@ 别名、/api 代理
└── tsconfig*.json            # TypeScript 配置
```

## 开发约定

- 页面与业务组件放在 `src/features/{domain}/` 下
- 新页面需要在 `src/App.tsx` 中注册路由
- API 调用统一走 `src/lib/api.ts`
- 类型定义统一维护在 `src/types/index.ts`，字段名需与后端 JSON 保持一致
- 状态管理以组件本地 `useState` / `useEffect` 为主，不引入 context 或第三方状态库
- UI 基础组件来自 `src/components/ui/*`，**不要直接修改这些生成文件**
- 不要直接引入 Radix primitives，优先复用现有 Shadcn 包装组件

## 与根项目的关系

- 根目录 `make frontend` 会在 `web/` 下执行 `bun install && bun run build`
- 根目录 `make build` 会先构建前端，再将 `web/dist/` 通过 `go:embed` 打包进 Go 二进制
- 如果只改动前端文档或页面逻辑，通常至少应重新执行 `bun run lint` 与 `bun run build`
