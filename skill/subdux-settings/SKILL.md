---
name: subdux-settings
description: Manage Subdux account setup data that supports subscription tracking, including categories, currencies, payment methods, and preferred currency. Use this skill when the user asks to organize, add, rename, reorder, clean up, or batch update Subdux settings. Do not use it for normal subscription CRUD; use the Subdux MCP server or API for subscriptions.
license: MIT
compatibility: Works with Open Agent Skills compatible clients that can read files and run Python 3 scripts with network access to a Subdux instance.
---

# Subdux Settings

Use this skill only for account setup and supporting metadata:

- Categories
- Currencies
- Payment methods
- Preferred currency
- Unused setting cleanup and batch reordering

Do not use this skill for normal subscription management. Subscriptions should be managed through the Subdux MCP server or Subdux subscription API.

## Authentication

The script reads connection settings in this order:

1. Command line flags
2. Environment variables
3. Config file
4. Defaults

Supported environment variables:

- `SUBDUX_BASE_URL`
- `SUBDUX_API_KEY`
- `SUBDUX_CONFIG`

Default config path:

```text
~/.config/subdux/config.yaml
```

Config file shape:

```yaml
base_url: "http://127.0.0.1:8080"
api_key: null
api_key_env: "SUBDUX_API_KEY"
timeout_seconds: 15
```

Do not write API keys into the skill directory, repository files, generated examples, logs, or plan files. Prefer `SUBDUX_API_KEY` for secrets. If a config file contains `api_key`, it must be stored outside the skill and repository with restrictive file permissions.

## Workflow

1. For inspection, run:

```bash
python skill/subdux-settings/scripts/subdux_settings.py list
```

2. For a proposed settings change, create a JSON desired-state file and run:

```bash
python skill/subdux-settings/scripts/subdux_settings.py plan --desired desired-settings.json --out subdux-settings-plan.json
```

For unused category, currency, and payment-method cleanup, run:

```bash
python skill/subdux-settings/scripts/subdux_settings.py cleanup-unused --out subdux-cleanup-plan.json
```

3. Show the plan summary to the user before applying changes.

4. Apply only when the user explicitly asks to apply, update, write, or execute the plan:

```bash
python skill/subdux-settings/scripts/subdux_settings.py apply --plan subdux-settings-plan.json
```

The script defaults to planning behavior. Apply requires an explicit `apply` command.

## Desired State

A desired-state file may include any subset of these fields:

```json
{
  "preferred_currency": "CNY",
  "currencies": [
    { "code": "CNY", "symbol": "¥", "alias": "Chinese yuan" },
    { "code": "USD", "symbol": "$", "alias": "US dollar" }
  ],
  "categories": [
    { "name": "Streaming" },
    { "name": "Developer Tools" }
  ],
  "payment_methods": [
    { "name": "Alipay", "icon": "custom:alipay" },
    { "name": "Visa", "icon": "lg:visa" }
  ]
}
```

Array order is treated as desired display order. Existing items are matched case-insensitively by `code` for currencies and by `name` for categories/payment methods.

## Safety Rules

- Read/list/plan operations require a Subdux API key with `read` scope.
- Apply operations require a Subdux API key with `write` scope.
- Never delete or merge settings unless the user specifically asked for cleanup or provided a plan that includes deletions.
- Deletion may fail if a category, currency, or payment method is still used by subscriptions; report the server error instead of forcing changes.
- Do not manage administrator system settings with this skill.
- Do not bypass Subdux authorization by reading or writing the SQLite database directly.

## References

Read only when needed:

- `references/api.md` for REST endpoints, payloads, and credential precedence.
- `references/examples.md` for desired-state and command examples.
