import backendMessages from "./backend-messages"

const common = {
  "loading": "Loading...",
  "cancel": "Cancel",
  "close": "Close",
  "clear": "Clear",
  "back": "Back",
  "copy": "Copy",
  "copied": "Copied",
  "showPassword": "Show password",
  "hidePassword": "Hide password",
  "unauthorized": "Unauthorized",
  "requestFailed": "Request failed",
  "backendMessages": backendMessages,
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
