# React Frontend

## OVERVIEW

React 19 SPA — Vite + Shadcn/UI (new-york/zinc) + Tailwind CSS v4. Feature-folder structure. No state library — pure useState.

## STRUCTURE

```
web/
├── src/
│   ├── App.tsx                  # Router: ProtectedRoute / PublicRoute guards
│   ├── main.tsx                 # React entry
│   ├── index.css                # Tailwind v4 + Shadcn oklch theme vars
│   ├── components/ui/           # Shadcn primitives (DO NOT edit directly)
│   │   ├── badge, button, card, dialog, input, label, select, separator
│   ├── features/
│   │   ├── auth/                # login-page.tsx, register-page.tsx
│   │   ├── dashboard/           # dashboard-page.tsx (stats + subscription list)
│   │   └── subscriptions/       # subscription-card.tsx, subscription-form.tsx (dialog)
│   ├── lib/
│   │   ├── api.ts               # Fetch wrapper with JWT, auto-redirect on 401
│   │   └── utils.ts             # cn(), formatCurrency(), formatDate(), daysUntil()
│   └── types/
│       └── index.ts             # All interfaces (mirrors Go models)
├── components.json              # Shadcn config: new-york style, zinc base
├── vite.config.ts               # React + Tailwind plugins, /api proxy → :8080
├── package.json                 # Bun runtime, React 19, Vite 7
└── tsconfig.app.json            # Path alias: @/ → src/
```

## WHERE TO LOOK

| Task | Location | Pattern |
|------|----------|---------|
| New page | `src/features/{domain}/{name}-page.tsx` + route in `App.tsx` | Default export function component |
| New feature component | `src/features/{domain}/` | Compose Shadcn UI primitives |
| New Shadcn component | Run `bunx shadcn@latest add {name}` from `web/` | Auto-creates in `src/components/ui/` |
| API integration | `src/lib/api.ts` | `api.get<T>()`, `api.post<T>()`, `api.put<T>()`, `api.delete<T>()` |
| Add TypeScript type | `src/types/index.ts` | Must match Go model's json tags exactly |
| Theme/colors | `src/index.css` | oklch CSS variables (light + dark mode defined) |

## CONVENTIONS

### API layer (`lib/api.ts`)
- Token stored in `localStorage` as `"token"`
- All requests auto-attach `Authorization: Bearer <token>` header
- 401 response → `clearToken()` + redirect to `/login`
- 204 response → returns `undefined` (no JSON parse)
- Error shape from backend: `{ error: "message" }` — thrown as `new Error(data.error)`

### Routing (`App.tsx`)
- `ProtectedRoute` — checks `isAuthenticated()`, redirects to `/login`
- `PublicRoute` — checks `isAuthenticated()`, redirects to `/`
- `isAuthenticated()` only checks token existence (not expiry)
- Catch-all `*` redirects to `/`

### Component patterns
- Pages: `function XPage()` with local useState for all state
- subscription-form.tsx: Dialog-based form, receives `onSubmit` callback + optional `subscription` for edit mode
- subscription-card.tsx: Inline display with edit/delete actions

### Styling
- Tailwind v4 via `@tailwindcss/vite` plugin (no tailwind.config)
- Shadcn: new-york variant, zinc palette, oklch color space
- All theme vars in `index.css` under `:root` and `.dark`
- Dark mode vars defined but no toggle mechanism implemented

## ANTI-PATTERNS

- **NEVER edit** files in `src/components/ui/` — Shadcn auto-generated, will be overwritten
- **NEVER import** Radix primitives directly — use Shadcn wrappers
- **No `useEffect` for data fetching** — current pattern uses `useEffect` + `useState` (no SWR/React Query)
- **No `useContext`** — everything is prop-drilled or local state
