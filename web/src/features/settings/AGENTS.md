# Settings Feature

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## OVERVIEW

User settings for display preferences, theme/language, currencies, payment methods, categories, notification channels/templates/policies/logs, account email/password/logout, API keys, audit events, and version/about information. The page is a single tabbed route with lazy tab content and feature hooks for heavier state.

## STRUCTURE

```
settings/
├── settings-page.tsx                         # Tab container, theme/display prefs, lazy tab mounting
├── settings-general-tab.tsx                  # Theme, color scheme, display options, language
├── settings-payment-tab.tsx                  # Currency prefs, currencies, payment methods, categories
├── hooks/use-settings-payment.ts             # Payment/currency/category state and mutations
├── settings-notification-tab.tsx             # Notification tab container
├── notification-channel-form/                # Channel-specific config fields and helpers
├── notification-channel-list.tsx             # Channel table/actions
├── notification-template-section/            # Template list/editor/preview dialog
├── notification-policy-section.tsx           # Days-before policy
├── notification-log-list.tsx                 # Delivery logs
├── settings-account-tab.tsx                  # Email change, password change, auth methods, logout
├── hooks/use-settings-account.ts             # Account/password/email-change state
├── settings-account-transfer-section.tsx     # Import/export UI
├── hooks/use-settings-account-transfer.ts    # Import/export state and file handling
├── totp-section.tsx, totp-setup-dialog.tsx   # TOTP setup/disable
├── passkey-section.tsx                       # Passkey registration/deletion
├── oidc-section.tsx                          # OIDC connection management
├── settings-apikey-tab.tsx                   # API key CRUD and MCP config snippets
├── settings-audit-tab.tsx                    # User audit events
└── settings-about-tab.tsx                    # Version/build information
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add settings tab | `settings-page.tsx` | Extend `SettingsTab`, `isSettingsTab`, trigger, lazy content |
| Display/theme/language | `settings-general-tab.tsx`, `lib/theme.ts`, `lib/display-preferences.ts` | Keep local-storage preference behavior stable |
| Currencies/payment/categories | `settings-payment-tab.tsx`, `hooks/use-settings-payment.ts` | Reorder and default behavior lives in hook |
| Notification channel | `notification-channel-form/`, `notification-channel-list.tsx` | Add config fields, validation hints, tests/backend support |
| Notification templates | `notification-template-section/` | Keep preview and variable behavior aligned with backend templates |
| Account/email/password | `settings-account-tab.tsx`, `hooks/use-settings-account.ts` | Human-session-only backend routes |
| Import/export UI | `settings-account-transfer-section.tsx`, `hooks/use-settings-account-transfer.ts` | File upload/download and toast feedback |
| TOTP/passkey/OIDC | `totp-section.tsx`, `passkey-section.tsx`, `oidc-section.tsx` | Keep WebAuthn/OIDC flow calls via `lib/api.ts` |
| API keys/MCP snippets | `settings-apikey-tab.tsx` | Include required MCP headers and key-kind behavior |
| Audit events | `settings-audit-tab.tsx` | Human-session-only endpoint |

## CONVENTIONS

### Tabs

- `settings-page.tsx` owns tab state, visited-tab lazy mounting, theme/display preferences, and version fetch.
- Current tabs: general, payment, notification, account, apikey, audit, about.
- Keep tab labels and all user-facing strings translated in `settings.ts` for en, zh-CN, and ja.

### State And Hooks

- Use local `useState`/`useEffect` and focused hooks.
- Keep payment/account/transfer complexity in hooks rather than expanding `settings-page.tsx`.
- For active-only data fetching, pass an `active` flag like the current notification/API-key/audit/account/payment patterns.

### Forms

- Use existing UI primitives and dialog patterns.
- Keep form state controlled.
- Use toasts for user-visible mutation success/failure where nearby settings code already does.
- Do not add form libraries for ordinary settings forms.

### Notification Channels

- Channel-specific fields live under `notification-channel-form/`.
- Shared config helpers/types/constants stay in that folder's helper files.
- New channels need backend validation/delivery/log behavior, settings form support, translations, and test coverage.

### API Keys And MCP

- API-key management is a human-session surface.
- Preserve REST vs MCP key-kind/scoping semantics.
- MCP snippets must include `X-API-Key`, `Content-Type: application/json`, and `Accept: application/json`.
- Do not expose admin/export/account-management affordances through MCP UI copy.

### Account Security

- Email change, password change, passkeys, OIDC connections, API keys, audit, and export are human-session-only backend routes.
- Keep destructive or sensitive actions explicit and reversible where possible.
- Do not store or display secrets beyond the one-time API-key reveal behavior already established.

## TESTING

For settings frontend changes:

```bash
cd web
bun run lint
bun run build
```

Run `bun run test` when changing shared helpers, formatting, safe-link behavior, passkey error mapping, currency/preset helpers, or other tested `lib` behavior. Backend-coupled settings changes should also run targeted Go tests for the corresponding API/service paths.

## ANTI-PATTERNS

- Adding settings state to a global store or context.
- Fetching authenticated data outside `lib/api.ts`.
- Adding untranslated settings text.
- Clearing secrets or API keys through ambiguous empty fields.
- Editing `components/ui` for settings-only layout tweaks.
- Expanding MCP or API-key UI claims beyond the backend's actual contract.
