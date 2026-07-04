# Settings Feature

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

User settings for theme/display/language, currencies/payment methods/categories, notification channels/templates/policies/logs, account security, API keys, audit events, and version/about information. The route is a single tabbed page with lazy tab mounting and focused hooks for heavier account/payment/import-export state.

## STRUCTURE

```text
settings/
├── settings-page.tsx                         # Tab container, visited-tab lazy mounting, version fetch
├── settings-general-tab.tsx                  # Theme, color scheme, display options, language
├── settings-payment-tab.tsx                  # Currency prefs, currencies, payment methods, categories
├── category-management.tsx                   # Category CRUD/reorder UI
├── payment-method-management.tsx             # Payment-method CRUD/reorder UI
├── hooks/use-settings-payment.ts             # Payment/currency/category state and mutations
├── settings-notification-tab.tsx             # Notification tab container
├── notification-channel-form.tsx             # Channel dialog entry point
├── notification-channel-form/                # Channel-specific fields, config helpers, secret inputs
├── notification-channel-list.tsx             # Channel table/actions
├── notification-template-section.tsx         # Template section entry point
├── notification-template-section/            # Template dialog helpers and constants
├── notification-policy-section.tsx           # Days-before policy
├── notification-log-list.tsx                 # Delivery logs
├── settings-account-tab.tsx                  # Email/password/auth methods/logout
├── hooks/use-settings-account.ts             # Account/password/email-change state
├── settings-account-transfer-section.tsx     # Import/export UI
├── hooks/use-settings-account-transfer.ts    # Import/export state and file handling
├── settings-account-import-types.ts          # Import/export response/form types
├── totp-section.tsx, totp-setup-dialog.tsx   # TOTP setup/disable
├── passkey-section.tsx                       # Passkey registration/deletion
├── oidc-section.tsx                          # OIDC connection management
├── settings-apikey-tab.tsx                   # API key CRUD and MCP snippets
├── settings-audit-tab.tsx                    # User audit events
└── settings-about-tab.tsx                    # Version/build/latest-release information
```

## WHERE TO LOOK

| Task | Start here | Notes |
|------|------------|-------|
| Add settings tab | `settings-page.tsx` | Extend `SettingsTab`, `isSettingsTab`, tab trigger, lazy content |
| Theme/display/language | `settings-general-tab.tsx`, `src/lib/theme.ts`, `src/lib/display-preferences.ts` | Keep local-storage behavior stable |
| Currencies/payment/categories | `settings-payment-tab.tsx`, `hooks/use-settings-payment.ts` | Reorder/default logic lives in the hook |
| Notification channel UI | `notification-channel-form.tsx`, `notification-channel-form/` | Add config fields, secret handling, translations, backend support |
| Notification templates | `notification-template-section.tsx`, `notification-template-section/` | Keep preview variables and validation aligned with backend templates |
| Account/email/password | `settings-account-tab.tsx`, `hooks/use-settings-account.ts` | Human-session-only backend routes |
| Import/export UI | `settings-account-transfer-section.tsx`, `hooks/use-settings-account-transfer.ts` | File upload/download, toasts, human/API-key boundaries |
| TOTP/passkey/OIDC | `totp-section.tsx`, `passkey-section.tsx`, `oidc-section.tsx` | Keep WebAuthn/OIDC flow calls in `src/lib/api.ts` |
| API keys and MCP snippets | `settings-apikey-tab.tsx` | Preserve key-kind/scoping semantics and MCP request examples |
| Audit events | `settings-audit-tab.tsx` | Human-session-only endpoint |

## CONVENTIONS

### Tabs And State

- `settings-page.tsx` owns tab state, visited-tab lazy mounting, theme/display preferences, and lazy version fetch.
- Current tabs: `general`, `payment`, `notification`, `account`, `apikey`, `audit`, `about`.
- Keep account/payment/import-export complexity in hooks rather than expanding `settings-page.tsx`.
- For active-only fetches, pass an `active` flag like the current account/payment patterns.

### Forms

- Use existing UI primitives and dialog patterns.
- Keep form state controlled.
- Use toasts for user-visible mutation feedback where neighboring settings code already does.
- Do not add form libraries for ordinary settings forms.

### Notification Channels

- Channel-specific fields live under `notification-channel-form/`.
- Shared types/constants/utils for channel dialogs stay in that folder.
- New channels need backend validation/delivery/log behavior, settings form support, translations, and tests.

### Account Security And API Keys

- Email change, password change, passkeys, OIDC connections, API keys, audit, and export are human-session-only backend surfaces.
- Preserve REST vs MCP key-kind/scoping semantics.
- MCP snippets must include `X-API-Key`, `Content-Type: application/json`, and `Accept: application/json`.
- Do not store or redisplay secrets beyond the one-time API-key reveal contract already established.

## TESTING

```bash
cd web
bun run lint
bun run build
```

Run `bun run test` when changing shared helpers or other tested frontend utilities. Backend-coupled settings changes should also run targeted Go tests for the corresponding API/service paths.
