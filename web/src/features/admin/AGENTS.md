# Admin Feature

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

Admin console for user management, system settings, SMTP, OIDC/authentication, exchange rates, background tasks, audit events, backup/restore, and reauth-gated sensitive admin actions. The route is guarded by `AdminRoute` in `App.tsx`; feature state is centralized in `hooks/use-admin-page-state.ts`.

## STRUCTURE

```text
admin/
├── admin-page.tsx                         # Lazy tab container and tab navigation
├── hooks/use-admin-page-state.ts          # Fetching, mutations, backup/restore, reauth, save flows
├── hooks/admin-settings-form.ts           # Admin settings form state/builders/mappers
├── hooks/admin-settings-form.test.ts      # Form-state regression tests
├── hooks/backup-destinations.ts           # Destination config parse/build helpers, secret masking
├── hooks/backup-destinations.test.ts      # Destination config helper tests
├── hooks/backup-run.ts                    # Backup run result summarization (delivery/retention/bookkeeping)
├── hooks/backup-run.test.ts               # Backup run summary tests
├── admin-users-tab.tsx                    # User list, create, role/status changes, disable auth factors, delete
├── reauth-dialog.tsx                      # Step-up prompt for sensitive admin mutations
├── admin-settings-tab.tsx                 # General site/security/image/MCP/audit/proxy settings
├── admin-settings-general-section.tsx     # Site/upload/MCP/audit fields
├── admin-settings-registration-section.tsx # Registration and auth-entry settings
├── admin-settings-ssrf-section.tsx        # Outbound SSRF controls and test surface
├── admin-settings-proxy-section.tsx       # System proxy settings
├── admin-settings-smtp-tab.tsx            # SMTP config and test delivery
├── admin-settings-smtp-section.tsx        # Core SMTP fields
├── admin-settings-smtp-advanced-fields.tsx
├── admin-settings-oidc-tab.tsx            # OIDC/authentication config
├── admin-settings-oidc-section.tsx
├── admin-settings-oidc-advanced-fields.tsx
├── admin-exchange-rates-tab.tsx           # Exchange source/API key/status/refresh
├── admin-background-tasks-tab.tsx         # Background task status and refresh
├── admin-audit-tab.tsx                    # Admin audit-event view
├── admin-backup-tab.tsx                   # Multi-destination backup (local/S3/WebDAV), run/test, local backups, restore upload
├── admin-backup-destinations-section.tsx  # Destination list, add/edit/delete, connectivity test
├── admin-backup-destination-form-fields.tsx # Type-switched local/S3/WebDAV config fields
├── admin-backup-format.ts                 # Byte/date formatting shared by the backup tab and its sections
├── admin-loading-skeleton.tsx             # Initial loading state
└── admin-settings-types.ts                # Admin settings UI types
```

## WHERE TO LOOK

| Task | Start here | Notes |
|------|------------|-------|
| Add admin tab | `admin-page.tsx` | Extend `AdminTab`, `isAdminTab`, tab trigger, lazy content |
| Add admin state/API call | `hooks/use-admin-page-state.ts` | Keep fetch/mutation logic out of tab render components |
| Change settings form shape | `hooks/admin-settings-form.ts`, `admin-settings-types.ts` | Update load/save mapping, defaults, and translations together |
| User management | `admin-users-tab.tsx`, `reauth-dialog.tsx` | Create user, role/status toggle, factor disable, delete |
| General/system settings | `admin-settings-tab.tsx`, related section files | Site, registration, upload, MCP, audit, SSRF, proxy |
| SMTP settings | `admin-settings-smtp-tab.tsx`, `admin-settings-smtp-section.tsx` | Configured-secret flags and test recipient behavior matter |
| OIDC settings | `admin-settings-oidc-tab.tsx` | Issuer/client/secret/scopes/advanced endpoints |
| Exchange rates | `admin-exchange-rates-tab.tsx` | Source/API key/status/refresh |
| Background tasks | `admin-background-tasks-tab.tsx` | Task monitor display and manual refresh |
| Audit events | `admin-audit-tab.tsx` | Admin audit endpoint |
| Backup/restore | `admin-backup-tab.tsx`, `hooks/backup-destinations.ts`, `hooks/backup-run.ts` | Destination CRUD/test (reauth-gated), include-assets option, local backups, restore confirmation |

## CONVENTIONS

### Route And Access

- `AdminRoute` in `App.tsx` checks authentication and `isAdmin()`.
- The backend also enforces admin JWT routes; do not rely on UI hiding as authorization.
- Sensitive admin mutations may require reauth. Keep UI prompts aligned with backend operation names and ticket flow.

### State And Tabs

- `useAdminPageState` owns initial loading, users, settings form, exchange status, background tasks, SMTP test state, backup/restore state, and reauth state.
- Keep tab components mostly presentational: props in, callbacks out.
- Tabs are lazy-loaded and mounted after first visit through `visitedTabs`.
- Current tabs: `users`, `settings`, `smtp`, `auth`, `exchange-rates`, `background-tasks`, `audit`, `backup`.

### Secrets And Settings

- Configured secrets use `*_configured` flags from the backend.
- Empty input usually means "keep existing secret", not "clear it", unless the UX explicitly supports clearing.
- Do not render secret values returned by the backend; show configured state instead.
- When adding a settings field, update form state, load/save mapping, relevant TypeScript DTOs, and translations together.

### UX

- Keep controls dense and operational. Admin pages are management surfaces, not marketing pages.
- Use existing tabs, sections, switches, inputs, selects, dialogs, and icon buttons.
- Refetch or update state deliberately after mutations; do not add optimistic behavior where backend settings may normalize or reject values.

## TESTING

```bash
cd web
bun run lint
bun run build
```

Also run `bun run test` when DTO-sensitive behavior or tested form helpers change. Backend admin changes need matching `go test ./internal/api ./internal/service/...` coverage where feasible.
