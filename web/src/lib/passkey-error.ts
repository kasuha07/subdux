type TranslateFn = (key: string) => string

export function getPasskeyErrorMessage(error: unknown, t: TranslateFn, fallbackKey: string): string {
  const key = getPasskeyErrorKey(error)
  if (key) {
    return t(key)
  }

  if (error instanceof Error && error.message.trim() !== "") {
    return error.message
  }

  return t(fallbackKey)
}

function getPasskeyErrorKey(error: unknown): string | null {
  const name = getErrorName(error)
  switch (name) {
    case "NotAllowedError":
      return "common.passkeyErrors.notAllowed"
    case "NotSupportedError":
      return "common.passkeyErrors.notSupported"
    case "InvalidStateError":
      return "common.passkeyErrors.invalidState"
    case "SecurityError":
      return "common.passkeyErrors.security"
    case "AbortError":
      return "common.passkeyErrors.aborted"
    case "NotFoundError":
      return "common.passkeyErrors.notFound"
    default:
      break
  }

  if (error instanceof Error) {
    const message = error.message.toLowerCase()
    if (message.includes("timed out or was not allowed")) {
      return "common.passkeyErrors.notAllowed"
    }
    if (message.includes("notallowederror")) {
      return "common.passkeyErrors.notAllowed"
    }
  }

  return null
}

function getErrorName(error: unknown): string {
  if (error instanceof DOMException) {
    return error.name
  }
  if (error && typeof error === "object" && "name" in error && typeof error.name === "string") {
    return error.name
  }
  return ""
}
