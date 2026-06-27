# Admin Feature

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## OVERVIEW

Admin console for user management, system settings, SMTP, proxy, OIDC/authentication, exchange rates, statistics, background tasks, audit events, and backup/restore. The route is guarded by `AdminRoute` in `App.tsx`; feature state is centralized in `hooks/use-admin-page-state.ts`.

## STRUCTURE

```
admin/
├── admin-page.tsx                         # Lazy tab container and tab navigation
├── hooks/use-admin-page-state.ts          # Fetching, mutations, form state, backup/restore
├── admin-users-tab.tsx                    # User list, create, role/status changes, delete
├── admin-settings-tab.tsx                 # General site/security/image/MCP/audit settings
├── admin-settings-general-section.tsx     # Site, registration, upload, MCP, audit fields
├── admin-settings-smtp-tab.tsx            # SMTP config and test delivery
├── admin-settings-smtp-advanced-fields.tsx
├── admin-settings-proxy-tab.tsx           # System proxy config
├── admin-settings-proxy-section.tsx
├── admin-settings-oidc-tab.tsx            # OIDC/authentication config
├── admin-settings-oidc-section.tsx
├── admin-settings-oidc-advanced-fields.tsx
├── admin-exchange-rates-tab.tsx           # Exchange source/API key/status/refresh
├── admin-stats-tab.tsx                    # Admin stats
├── admin-background-tasks-tab.tsx         # Background task status and refresh
├── admin-audit-tab.tsx                    # Admin audit-event view
├── admin-backup-tab.tsx                   # Backup download and restore upload
├── admin-loading-skeleton.tsx             # Initial loading state
└── admin-settings-types.ts                # Admin settings UI types
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add admin tab | `admin-page.tsx` | Extend `AdminTab`, `isAdminTab`, tab trigger, lazy content |
| Add admin state/API call | `hooks/use-admin-page-state.ts` | Keep fetch/mutation logic out of tab render components |
| User management | `admin-users-tab.tsx` | Create user, role/status toggle, delete |
| General/system settings | `admin-settings-tab.tsx`, `admin-settings-general-section.tsx` | Site, registration, upload, MCP, audit |
| SMTP settings | `admin-settings-smtp-tab.tsx` | Configured-secret flags and test recipient behavior matter |
| Proxy settings | `admin-settings-proxy-tab.tsx` | System proxy URL may be configured without exposing secret value |
| OIDC settings | `admin-settings-oidc-tab.tsx` | Issuer/client/secret/scopes/advanced endpoints |
| Exchange rates | `admin-exchange-rates-tab.tsx` | Source/API key/status/refresh |
| Background tasks | `admin-background-tasks-tab.tsx` | Task monitor display and manual refresh |
| Audit events | `admin-audit-tab.tsx` | Admin audit endpoint |
| Backup/restore | `admin-backup-tab.tsx` | Include-assets option and restore confirmation |

## CONVENTIONS

### Route And Access

- `AdminRoute` in `App.tsx` checks authentication and `isAdmin()`.
- The backend also enforces admin JWT routes; do not rely on UI hiding as authorization.
- Admin status comes from the cached user object managed by `lib/api.ts`.

### State

- `useAdminPageState` owns initial loading, users, stats, settings form, exchange status, background tasks, SMTP test state, and backup/restore state.
- Keep tab components mostly presentational: props in, callbacks out.
- When adding a settings field, update `AdminSettingsFormState`, `createSettingsForm`, save payload mapping, relevant TypeScript DTOs, and translations.

### Tabs

- Tabs are lazy-loaded and only mounted after first visit through `visitedTabs`.
- Current tabs: users, settings, smtp, proxy, auth, exchange-rates, stats, background-tasks, audit, backup.
- Keep tab labels translated through `admin.ts` locale files.

### Secret Fields

- Configured secrets use `*_configured` flags from the backend.
- Empty input should usually mean "keep existing secret", not "clear it", unless the UX explicitly supports clearing.
- Do not render secret values returned by the backend; show configured state instead.

### UX

- Keep controls dense and operational. Admin pages are management surfaces, not marketing pages.
- Use existing tabs, sections, switches, inputs, selects, and icon buttons.
- Refetch or update state deliberately after mutations; do not add optimistic behavior where backend settings may normalize or reject values.

## TESTING

For admin frontend changes:

```bash
cd web
bun run lint
bun run build
```

Also run `bun run test` when shared `lib` helpers or DTO-sensitive behavior changes. Backend admin changes need matching `go test ./internal/api ./internal/service` coverage where possible.

## ANTI-PATTERNS

- Adding admin-only behavior that is only protected by UI checks.
- Spreading settings fetch/save logic across tab components.
- Clearing configured secrets from empty form fields.
- Adding untranslated tab labels, field labels, or toast text.
- Editing `components/ui` for admin-specific presentation.
