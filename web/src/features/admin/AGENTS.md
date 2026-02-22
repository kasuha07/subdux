# Admin Feature

**Generated:** 2026-02-22 13:12 UTC  
**Commit:** fdfaf8c  
**Branch:** main

## OVERVIEW

16 files implementing admin panel: user management, stats, system settings (SMTP, OIDC), exchange rates, backup/restore. Tab-based UI with role guard.

## STRUCTURE

```
admin/
├── admin-page.tsx                       # Tab container (5 tabs) + AdminRoute guard
├── admin-users-tab.tsx                  # User list, role management, delete
├── admin-stats-tab.tsx                  # Dashboard stats, metrics
├── admin-settings-tab.tsx               # System settings container
├── admin-settings-general-section.tsx   # Site name, currency, timezone
├── admin-settings-smtp-tab.tsx          # Email server config
├── admin-settings-smtp-advanced-fields.tsx
├── admin-settings-oidc-tab.tsx          # OAuth provider config
├── admin-settings-oidc-section.tsx
├── admin-settings-oidc-advanced-fields.tsx
├── admin-exchange-rates-tab.tsx         # Currency conversion rates
├── admin-backup-tab.tsx                 # Export/import data
├── admin-loading-skeleton.tsx           # Loading state
├── admin-settings-types.ts              # TypeScript interfaces
└── hooks/                               # Admin-specific hooks
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add admin tab | `admin-page.tsx` | Add `<TabsContent>` + update tab list |
| User management | `admin-users-tab.tsx` | List, role change, delete |
| System settings | `admin-settings-tab.tsx` | Nested tabs for SMTP/OIDC |
| SMTP config | `admin-settings-smtp-tab.tsx` | Email server, auth, TLS |
| OIDC config | `admin-settings-oidc-tab.tsx` | Provider URL, client ID/secret |
| Exchange rates | `admin-exchange-rates-tab.tsx` | Currency conversion management |
| Backup/restore | `admin-backup-tab.tsx` | JSON export/import |

## CONVENTIONS

### Route Guard
- `AdminRoute` component in `App.tsx` checks `isAdmin()`
- Redirects non-admins to `/`
- Admin status cached in localStorage `"user"` object

### Tab Structure
- 5 top-level tabs: Users, Stats, Settings, Exchange Rates, Backup
- Settings tab has nested tabs (General, SMTP, OIDC)
- Each tab fetches its own data with `useEffect`

### Form Patterns
- Inline editing (no dialogs)
- Save buttons per section
- Advanced fields collapsed by default (accordion)

### Loading States
- `admin-loading-skeleton.tsx` for consistent loading UI
- Used across all tabs during data fetch

## ANTI-PATTERNS

- **No role-based UI hiding** — entire admin feature is route-guarded
- **No optimistic updates** — always refetch after mutations
- **No form validation library** — plain HTML5 validation
