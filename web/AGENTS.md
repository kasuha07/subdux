# React Frontend

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## OVERVIEW

React 19 SPA built with Vite 8, Bun, Tailwind v4, local Shadcn-style UI primitives, React Router 7, i18next, Sonner, and Vitest. The production build is embedded by the Go server through `frontend.go`.

The frontend is organized by feature folders and uses local component state plus feature hooks. API calls go through `src/lib/api.ts`, which owns access-token memory, refresh-cookie recovery, user cache, JSON parsing, localized backend errors, 401 handling, and upload helpers.

## STRUCTURE

```
web/
├── src/
│   ├── App.tsx                  # Lazy routes and Public/Protected/Admin guards
│   ├── main.tsx                 # React entry
│   ├── index.css                # Tailwind v4 and theme variables
│   ├── components/              # App-level components plus ui primitives
│   ├── components/ui/           # Shadcn-style primitives; avoid one-off edits
│   ├── features/
│   │   ├── auth/                # login, register, forgot/reset password
│   │   ├── dashboard/           # summary cards, filters, dashboard data hooks
│   │   ├── subscriptions/       # cards, form, detail drawer, icon picker
│   │   ├── actions/             # renewal/stale action center
│   │   ├── reports/             # analytics reports and panels
│   │   ├── calendar/            # calendar token/feed UI
│   │   ├── settings/            # user settings; see feature AGENTS.md
│   │   └── admin/               # admin console; see feature AGENTS.md
│   ├── hooks/                   # Shared app hooks such as site settings
│   ├── i18n/                    # en, zh-CN, ja translations by domain
│   ├── lib/                     # API client, brand icons, formatting, theme, safety helpers
│   └── types/                   # API DTOs split by domain, plus compatibility barrel
├── package.json                 # Bun scripts: dev, build, lint, test
├── vite.config.ts               # React + Tailwind plugins, /api proxy to Go server
└── tsconfig.app.json            # Path alias @/ -> src/
```

## WHERE TO LOOK

| Task | Location | Pattern |
|------|----------|---------|
| Add page/route | `src/features/{domain}/` + `src/App.tsx` | Lazy import plus route guard |
| Add API call | `src/lib/api.ts` consumer | Use `api.get/post/put/delete/uploadFile` |
| Add shared API type | `src/types/{domain}.ts` | Re-export from `src/types/index.ts` when broadly used |
| Add feature-local type | Closest `types.ts` or component file | Keep view/form-only types out of API DTOs |
| Add i18n text | `src/i18n/{en,zh-CN,ja}/{domain}.ts` | Keep keys aligned across all locales |
| Add setting UI | `src/features/settings/` | See `src/features/settings/AGENTS.md` |
| Add admin UI | `src/features/admin/` | See `src/features/admin/AGENTS.md` |
| Change brand icons | `src/lib/brand-icons.ts`, `src/lib/brand-icons/*` | Keep public icon helper API stable |
| Change theme/display prefs | `src/lib/theme.ts`, `src/lib/display-preferences.ts`, settings general tab | Keep local-storage behavior intentional |

## CONVENTIONS

### API Layer

- Use `api.get<T>()`, `api.post<T>()`, `api.put<T>()`, `api.delete<T>()`, or `api.uploadFile<T>()`.
- Do not bypass `lib/api.ts` for authenticated JSON calls.
- Access tokens are kept in memory; refresh uses the backend refresh cookie.
- `localStorage` stores the cached `"user"` object but the `"token"` key is intentionally cleared by current token handling.
- `401` responses trigger refresh where allowed, otherwise clear auth and redirect to `/login`.
- Backend error shape is `{ error: "message" }`; `lib/api.ts` localizes known backend errors.

### Routing

- `App.tsx` owns lazy routes for auth, dashboard, actions, reports, settings, calendar, and admin.
- `ProtectedRoute` requires `isAuthenticated()`.
- `PublicRoute` redirects already-authenticated users to `/`.
- `AdminRoute` requires both authentication and `isAdmin()`.
- Catch-all routes redirect to `/`.

### State And Data Fetching

- Use local `useState`, `useEffect`, `useRef`, and small feature hooks.
- Do not add global state libraries or React context for ordinary page state.
- Lazy-load heavier tabs/pages with `React.lazy` and `Suspense` where the current feature already does so.
- Keep data-fetching behavior close to the owning feature or hook.

### UI And Styling

- Use `src/components/ui/*` primitives and existing app components.
- Do not import Radix primitives directly in feature code.
- Tailwind v4 is configured through the Vite plugin; there is no Tailwind config file.
- Icons should usually come from `lucide-react` or existing brand-icon helpers.
- Keep text compact and translated. Avoid visible in-app descriptions of implementation details or keyboard shortcuts unless already part of the UX pattern.

### Types And i18n

- `src/types/index.ts` should stay a thin compatibility barrel.
- API DTO field names must match backend JSON tags exactly.
- Add or change user-facing copy in all three locales: `en`, `zh-CN`, and `ja`.
- Prefer domain translation files over unrelated common buckets.

### Brand Icons

- Keep exported API in `src/lib/brand-icons.ts` stable: `brandIcons`, `getBrandIcon`, `getBrandIconFromValue`.
- Keep `src/lib/brand-icons/specs.ts` as an aggregator.
- Add provider specs to the closest domain file under `src/lib/brand-icons/specs/`.
- Custom icons belong in `src/lib/brand-icons/custom/<slug>.ts`; values use `custom:<slug>`.

## TESTING

```bash
cd web
bun run lint
bun run build
bun run test
```

Frontend tests currently cover shared `lib` helpers. For UI-only changes without tests, verify with lint/build and inspect the rendered layout when a dev server is useful.

## ANTI-PATTERNS

- Editing `src/components/ui/*` for a one-off feature need.
- Importing Radix primitives directly.
- Duplicating auth, refresh, upload, or error handling outside `lib/api.ts`.
- Introducing global state libraries or context for routine feature state.
- Adding untranslated user-facing strings.
- Inlining large custom SVG icon definitions into aggregators.
