# Subdux Frontend

Subdux 前端基于 React 19 + Vite + TypeScript，作为 SPA 构建后会被嵌入后端二进制。

> 完整项目说明（后端、部署、环境变量）请查看根目录 [`README.md`](../README.md)。

## 技术栈

- React 19
- Vite 7
- TypeScript 5
- Tailwind CSS v4
- Shadcn/UI
- i18next

## 开发命令

```bash
cd web
bun install
bun dev
```

- 默认开发地址：`http://localhost:5173`
- `/api` 请求会代理到 `http://localhost:8080`

## 构建与检查

```bash
cd web
bun run build
bun run lint
bun run preview
```

## 目录约定

```text
web/src/
├── features/         # 业务模块（auth/dashboard/settings/admin/...）
├── components/ui/    # Shadcn 生成组件（不要手改）
├── lib/              # API 封装、工具函数、图标映射
└── types/            # 全局类型定义
```

## 开发注意事项

- API 调用统一走 `src/lib/api.ts`
- 不要直接修改 `src/components/ui/*`
- 新页面放在 `src/features/{domain}/` 下，并在 `src/App.tsx` 注册路由
