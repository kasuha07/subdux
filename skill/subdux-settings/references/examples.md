# Subdux Settings Examples

## Desired State

```json
{
  "preferred_currency": "CNY",
  "currencies": [
    { "code": "CNY", "symbol": "¥", "alias": "Chinese yuan" },
    { "code": "USD", "symbol": "$", "alias": "US dollar" },
    { "code": "JPY", "symbol": "¥", "alias": "Japanese yen" }
  ],
  "categories": [
    { "name": "Streaming" },
    { "name": "Developer Tools" },
    { "name": "Cloud Services" }
  ],
  "payment_methods": [
    { "name": "Alipay", "icon": "custom:alipay" },
    { "name": "WeChat Pay", "icon": "custom:wechatpay" },
    { "name": "Visa", "icon": "lg:visa" }
  ]
}
```

## Planning

```bash
python skill/subdux-settings/scripts/subdux_settings.py plan \
  --desired desired-settings.json \
  --out subdux-settings-plan.json
```

The plan file contains actions but no API key. Review it before applying.

## Cleanup Unused Settings

```bash
python skill/subdux-settings/scripts/subdux_settings.py cleanup-unused \
  --out subdux-cleanup-plan.json
```

This only creates a deletion plan for unused currencies, categories, and payment methods. Subdux may still reject a deletion during apply if the item becomes used before the plan is applied.

## Applying

```bash
python skill/subdux-settings/scripts/subdux_settings.py apply \
  --plan subdux-settings-plan.json
```

## One-off Connection Override

```bash
SUBDUX_BASE_URL="https://subdux.example.com" \
SUBDUX_API_KEY="sdx_xxx" \
python skill/subdux-settings/scripts/subdux_settings.py list
```

## Local Config

```bash
python skill/subdux-settings/scripts/subdux_settings.py write-config \
  --base-url http://127.0.0.1:8080 \
  --api-key-env SUBDUX_API_KEY
```

This writes connection metadata only. Keep the actual API key in `SUBDUX_API_KEY`.

The generated config is YAML by default:

```yaml
base_url: "http://127.0.0.1:8080"
api_key: null
api_key_env: "SUBDUX_API_KEY"
timeout_seconds: 15
```
