# Feature Folders

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

`web/src/features` owns route-level and domain-level frontend code. Each folder should keep its own page logic, local hooks, and feature-local components close together. Use the nearest child `AGENTS.md` when one exists.

## STRUCTURE

```text
features/
├── auth/             # Login, register, forgot/reset password, OIDC reauth callback
├── dashboard/        # Home route, filters, summary cards, dashboard hooks/helpers
├── subscriptions/    # Shared subscription cards, form, detail drawer, icon picker, hooks
├── actions/          # Renewal/stale action center page
├── reports/          # Analytics page and panels
├── calendar/         # Calendar feed/token page
├── settings/         # User settings feature; see settings/AGENTS.md
└── admin/            # Admin console feature; see admin/AGENTS.md
```

## WHERE TO LOOK

| Task | Folder | Notes |
|------|--------|-------|
| Login/register/reset/OIDC reauth | `auth/` | Public auth routes and callback flows |
| Dashboard home route | `dashboard/` | Summary cards, filters, amount utilities, hooks |
| Shared subscription UI | `subscriptions/` | Form, drawer, icon picker, recurrence/notification fields |
| Action center page | `actions/` | Renewal/stale action list and page wiring |
| Analytics reports | `reports/` | Report panels and report-specific display logic |
| Calendar feed UI | `calendar/` | Token/feed management and subscription rendering |
| User settings | `settings/AGENTS.md` | Tabbed settings route |
| Admin console | `admin/AGENTS.md` | Admin route and tabs |

## CONVENTIONS

- Keep route/page ownership in the matching feature folder.
- Put feature-local hooks inside that feature when they are not broadly shared.
- Keep shared subscription presentation logic in `subscriptions/` instead of duplicating cards/forms across dashboard, actions, reports, or calendar.
- Use shared app primitives from `src/components` and authenticated API calls from `src/lib/api.ts`.
- Add or change user-facing text in all locale files.

## TESTING

```bash
cd web
bun run lint
bun run build
```

Run `bun run test` when feature work also changes shared tested helpers in `src/lib` or related utility modules.
