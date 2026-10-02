import { describe, expect, it } from "vitest"
import { safeMCPReturnPath } from "./mcp-login-return"

describe("MCP login return paths", () => {
  it("preserves only a local consent interaction", () => {
    const path = `/connect/mcp?request=${"A".repeat(43)}`
    expect(safeMCPReturnPath(path)).toBe(path)
  })

  it.each([null, "https://evil.example", "//evil.example", "/\\evil.example", "/connect/mcp?request=short", `/connect/mcp?request=${"A".repeat(43)}&next=https://evil.example`, `/connect/mcp?request=${"A".repeat(43)}#extra`])("rejects unsafe or malformed return paths: %s", (path) => {
    expect(safeMCPReturnPath(path)).toBeNull()
  })
})
