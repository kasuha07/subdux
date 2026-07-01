const common = {
  "loading": "Loading...",
  "cancel": "Cancel",
  "close": "Close",
  "unauthorized": "Unauthorized",
  "requestFailed": "Request failed",
  "backendErrors": {
    "maxNotificationChannels": "You can enable at most 3 notification channels",
    "smtpRateLimited": "SMTP send rate limit reached. Please wait before trying again.",
    "reauthRequired": "Re-authentication failed. Please try again.",
    "noPasskeyRegistered": "No passkey is registered for your account. Add one in Settings, or use your password."
  },
  "passkeyErrors": {
    "notAllowed": "Passkey request was cancelled or timed out",
    "notSupported": "Passkey is not supported on this device or browser",
    "invalidState": "This passkey is already registered on this device",
    "security": "Passkey is unavailable on this site. Check domain and HTTPS settings",
    "aborted": "Passkey request was interrupted",
    "notFound": "No matching passkey was found"
  }
} as const

export default common
