export function safeHref(value?: string | null): string | undefined {
  const trimmed = value?.trim()
  if (!trimmed || hasUnsafeURLCharacter(trimmed)) {
    return undefined
  }

  try {
    const parsed = new URL(trimmed)
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return undefined
    }
    if (!parsed.hostname) {
      return undefined
    }
    return parsed.href
  } catch {
    return undefined
  }
}

function hasUnsafeURLCharacter(value: string): boolean {
  for (let index = 0; index < value.length; index++) {
    const code = value.charCodeAt(index)
    if (code <= 0x20 || code === 0x7f) {
      return true
    }
  }
  return false
}
