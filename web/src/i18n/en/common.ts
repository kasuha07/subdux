const common = {
  "loading": "Loading...",
  "unauthorized": "Unauthorized",
  "requestFailed": "Request failed",
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
