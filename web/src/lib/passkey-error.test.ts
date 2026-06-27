import { describe, expect, it, vi } from "vitest"

import { getPasskeyErrorMessage } from "@/lib/passkey-error"

// A translate fn that echoes its key so we can assert which message key was chosen.
const echo = (key: string): string => key

describe("getPasskeyErrorMessage", () => {
  it.each([
    ["NotAllowedError", "common.passkeyErrors.notAllowed"],
    ["NotSupportedError", "common.passkeyErrors.notSupported"],
    ["InvalidStateError", "common.passkeyErrors.invalidState"],
    ["SecurityError", "common.passkeyErrors.security"],
    ["AbortError", "common.passkeyErrors.aborted"],
    ["NotFoundError", "common.passkeyErrors.notFound"],
  ])("maps DOMException %s to %s", (name, expectedKey) => {
    const error = new DOMException("boom", name)
    expect(getPasskeyErrorMessage(error, echo, "fallback")).toBe(expectedKey)
  })

  it("maps a plain object with a known error name", () => {
    expect(getPasskeyErrorMessage({ name: "AbortError" }, echo, "fallback")).toBe(
      "common.passkeyErrors.aborted"
    )
  })

  it("matches the 'timed out or was not allowed' message heuristic", () => {
    const error = new Error("The operation timed out or was not allowed.")
    expect(getPasskeyErrorMessage(error, echo, "fallback")).toBe(
      "common.passkeyErrors.notAllowed"
    )
  })

  it("matches the 'notallowederror' message heuristic case-insensitively", () => {
    const error = new Error("Caught a NotAllowedError downstream")
    expect(getPasskeyErrorMessage(error, echo, "fallback")).toBe(
      "common.passkeyErrors.notAllowed"
    )
  })

  it("returns the raw error message for an unrecognised Error", () => {
    const error = new Error("something specific went wrong")
    expect(getPasskeyErrorMessage(error, echo, "fallback")).toBe(
      "something specific went wrong"
    )
  })

  it("uses the fallback key for an Error with a blank message", () => {
    const error = new Error("   ")
    expect(getPasskeyErrorMessage(error, echo, "fallback.key")).toBe("fallback.key")
  })

  it("uses the fallback key for a non-error value", () => {
    expect(getPasskeyErrorMessage("nope", echo, "fallback.key")).toBe("fallback.key")
    expect(getPasskeyErrorMessage(undefined, echo, "fallback.key")).toBe("fallback.key")
  })

  it("calls the translate fn rather than returning the key verbatim", () => {
    const t = vi.fn((key: string) => `translated:${key}`)
    const result = getPasskeyErrorMessage(new DOMException("x", "SecurityError"), t, "fb")
    expect(t).toHaveBeenCalledWith("common.passkeyErrors.security")
    expect(result).toBe("translated:common.passkeyErrors.security")
  })
})
