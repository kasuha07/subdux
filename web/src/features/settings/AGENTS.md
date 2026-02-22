# Settings Feature

**Generated:** 2026-02-22 13:12 UTC  
**Commit:** fdfaf8c  
**Branch:** main

## OVERVIEW

29 files implementing user settings: account, notifications (channels/templates/policies), payment methods, categories, TOTP/passkey/OIDC. Tab-based UI with dialog forms.

## STRUCTURE

```
settings/
├── settings-page.tsx                    # Tab container (4 tabs)
├── settings-account-tab.tsx             # Username, password, delete account
├── settings-notification-tab.tsx        # Channels, templates, policies, logs
├── settings-payment-tab.tsx             # Payment methods, categories
├── settings-general-tab.tsx             # General preferences
├── notification-channel-form/           # 9 files: multi-step form (type → config → test)
├── notification-channel-list.tsx        # Channel CRUD table
├── notification-template-section/       # 2 files: template editor
├── notification-policy-section.tsx      # Days-before rules
├── notification-log-list.tsx            # Delivery history
├── totp-section.tsx, totp-setup-dialog.tsx
├── passkey-section.tsx
├── oidc-section.tsx
├── payment-method-management.tsx
├── category-management.tsx
└── hooks/                               # 2 files: useNotificationChannels, useNotificationTemplates
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add settings tab | `settings-page.tsx` | Add `<TabsContent>` + route in parent |
| Notification channels | `notification-channel-form/` | Multi-step: type selection → config → test |
| Notification templates | `notification-template-section/` | Template editor with variables |
| 2FA setup | `totp-section.tsx`, `passkey-section.tsx` | QR code, recovery codes, WebAuthn |
| OAuth setup | `oidc-section.tsx` | Provider config, callback URL |
| Payment methods | `payment-method-management.tsx` | CRUD with default selection |
| Categories | `category-management.tsx` | CRUD with color picker |

## CONVENTIONS

### Tab Structure
- `settings-page.tsx` uses Shadcn `<Tabs>` with 4 tabs
- Each tab is a separate `*-tab.tsx` file
- Tabs use local `useState` for data fetching

### Form Patterns
- Dialog-based forms (Shadcn `<Dialog>`)
- Multi-step forms use `useState` for step tracking
- `onSubmit` callbacks passed from parent
- Edit mode: pass existing model as prop

### Notification Channel Form
- **Step 1:** Type selection (email, webhook, ServerChan, Bark, Telegram)
- **Step 2:** Config form (type-specific fields)
- **Step 3:** Test delivery (optional)

### Hooks
- `useNotificationChannels()` — fetch + CRUD operations
- `useNotificationTemplates()` — fetch + CRUD operations
- Both return `{ data, loading, error, create, update, delete }`

## ANTI-PATTERNS

- **No nested routing** — all tabs in single page component
- **No form libraries** — plain controlled inputs with `useState`
- **No global state** — each tab fetches its own data
