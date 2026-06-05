# Subdux Settings API Reference

The bundled script uses Subdux REST endpoints with the `X-API-Key` header. It does not access the database directly.

## Authentication

Headers:

```http
X-API-Key: sdx_xxx
Content-Type: application/json
```

API keys are created in Subdux user settings. Read-only operations need `read` scope. Mutating operations need `write` scope.

Credential precedence:

1. `--base-url`, `--api-key`, `--config`
2. `SUBDUX_BASE_URL`, `SUBDUX_API_KEY`, `SUBDUX_CONFIG`
3. YAML config file values
4. Default base URL `http://127.0.0.1:8080`

If `api_key_env` exists in the config file, that named environment variable is checked after `SUBDUX_API_KEY` and before `config.api_key`.

Default config path:

```text
~/.config/subdux/config.yaml
```

Config example:

```yaml
base_url: "http://127.0.0.1:8080"
api_key: null
api_key_env: "SUBDUX_API_KEY"
timeout_seconds: 15
```

The script also accepts `.json` config files for compatibility when `--config` points to one, but YAML is the recommended format.

## Endpoints

Preferences:

```text
GET /api/preferences/currency
PUT /api/preferences/currency
```

Payload:

```json
{ "preferred_currency": "CNY" }
```

Currencies:

```text
GET /api/currencies
POST /api/currencies
PUT /api/currencies/:id
PUT /api/currencies/reorder
DELETE /api/currencies/:id
```

Create payload:

```json
{ "code": "CNY", "symbol": "¥", "alias": "Chinese yuan", "sort_order": 0 }
```

Update payload:

```json
{ "symbol": "¥", "alias": "Chinese yuan", "sort_order": 0 }
```

Reorder payload:

```json
[
  { "id": 1, "sort_order": 0 },
  { "id": 2, "sort_order": 1 }
]
```

Categories:

```text
GET /api/categories
POST /api/categories
PUT /api/categories/:id
PUT /api/categories/reorder
DELETE /api/categories/:id
```

Create payload:

```json
{ "name": "Streaming", "display_order": 0 }
```

Payment methods:

```text
GET /api/payment-methods
POST /api/payment-methods
PUT /api/payment-methods/:id
PUT /api/payment-methods/reorder
DELETE /api/payment-methods/:id
```

Create payload:

```json
{ "name": "Alipay", "icon": "custom:alipay", "sort_order": 0 }
```

Payment method icons must pass Subdux icon validation. Empty strings, emoji, managed `file:` icons, and supported icon identifiers are accepted by the server. Common examples include `custom:alipay`, `custom:wechatpay`, `lg:visa`, `lg:mastercard`, and `lg:paypal`.

## Script Commands

```bash
python scripts/subdux_settings.py list
python scripts/subdux_settings.py plan --desired desired-settings.json --out plan.json
python scripts/subdux_settings.py cleanup-unused --out cleanup-plan.json
python scripts/subdux_settings.py apply --plan plan.json
python scripts/subdux_settings.py write-config --base-url http://127.0.0.1:8080
```

Use `--json` for machine-readable command output.
