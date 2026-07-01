const common = {
  "loading": "加载中...",
  "cancel": "取消",
  "close": "关闭",
  "unauthorized": "未授权",
  "requestFailed": "请求失败",
  "backendErrors": {
    "maxNotificationChannels": "最多只能启用 3 个通知渠道",
    "smtpRateLimited": "SMTP 发信速率已达上限，请稍后再试",
    "reauthRequired": "重新验证失败，请重试。",
    "noPasskeyRegistered": "您的账户尚未注册通行密钥。请在设置中添加，或使用密码。"
  },
  "passkeyErrors": {
    "notAllowed": "Passkey 请求已取消或超时",
    "notSupported": "当前设备或浏览器不支持 Passkey",
    "invalidState": "该 Passkey 已在此设备上注册",
    "security": "当前站点无法使用 Passkey，请检查域名和 HTTPS 配置",
    "aborted": "Passkey 请求被中断",
    "notFound": "未找到可用的 Passkey"
  }
} as const

export default common
