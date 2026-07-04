# React Frontend

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

React 19 SPA built with Vite 8, Bun, Tailwind v4, local Shadcn-style UI primitives, React Router 7, i18next, Sonner, and Vitest. The production build is embedded by the Go server through `frontend.go`.

This file is the frontend index. Read the nearest child `AGENTS.md` before changing a feature folder.

## FRONTEND AGENTS INDEX

| Scope | File | Use when |
|------|------|----------|
| Frontend index | `web/AGENTS.md` | Cross-feature UI work, shared frontend utilities, choosing the right child guide |
| Feature index | `web/src/features/AGENTS.md` | Picking the right route/feature folder |
| Settings feature | `web/src/features/settings/AGENTS.md` | User settings, account, API keys, audit, imports/exports |
| Admin feature | `web/src/features/admin/AGENTS.md` | Admin users/settings/backup/reauth flows |

## STRUCTURE

```text
web/
├── src/
│   ├── App.tsx                  # Lazy routes, guards, route preloading
│   ├── main.tsx                 # React entry
│   ├── index.css                # Tailwind v4 and theme variables
│   ├── components/              # App-level components plus ui primitives
│   ├── components/ui/           # Local UI primitives; avoid one-off edits
│   ├── features/                # Route and domain folders; see child AGENTS
│   ├── hooks/                   # Shared app hooks such as site settings/title
│   ├── i18n/                    # en, zh-CN, ja translations by domain
│   ├── lib/                     # API client, formatting, theme, brand icons, safety helpers
│   └── types/                   # API DTOs split by domain plus compatibility barrel
├── package.json                 # Bun scripts: dev, build, lint, test
├── vite.config.ts               # React + Tailwind plugins, /api proxy to Go server
└── tsconfig.app.json            # Path alias @/ -> src/
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add page or route | `src/features/{domain}/` + `src/App.tsx` | Lazy import plus route guard |
| Choose the owning feature | `src/features/AGENTS.md` | Feature-level folder map |
| Add authenticated API call | `src/lib/api.ts` or its consumers | Do not bypass `lib/api.ts` |
| Add shared API type | `src/types/{domain}.ts` | Keep `src/types/index.ts` a thin barrel |
| Add i18n text | `src/i18n/{en,zh-CN,ja}/{domain}.ts` | Keep all locales aligned |
| Change theme/display prefs | `src/lib/theme.ts`, `src/lib/display-preferences.ts`, settings general tab | Keep local-storage behavior intentional |
| Change brand icons | `src/lib/brand-icons.ts`, `src/lib/brand-icons/*` | Keep public helper API stable |

## FRONTEND-WIDE RULES

- Use `api.get<T>()`, `api.post<T>()`, `api.put<T>()`, `api.delete<T>()`, or `api.uploadFile<T>()`.
- Do not bypass `src/lib/api.ts` for authenticated JSON flows. It owns access-token memory, refresh-cookie recovery, 401 handling, and known backend error localization.
- Keep route ownership in `src/App.tsx`: `ProtectedRoute`, `PublicRoute`, and `AdminRoute` are the access boundaries.
- Prefer local component state and small feature hooks. Do not introduce global state libraries or React context for ordinary page state.
- Use existing `src/components/ui/*` primitives and app components; do not import Radix primitives directly in feature code.
- Keep user-facing copy translated in `en`, `zh-CN`, and `ja`.

## TESTING

```bash
cd web
bun run lint
bun run build
bun run test
```

Frontend tests currently concentrate in shared helpers. For UI-only changes without direct tests, still run lint/build and inspect the route in a dev server when layout or interaction risk is non-trivial.
