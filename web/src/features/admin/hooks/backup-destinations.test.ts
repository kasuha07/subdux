import { describe, expect, it } from "vitest"

import type { BackupDestination } from "@/types"

import {
  buildLocalConfig,
  buildS3Config,
  buildWebDAVConfig,
  DESTINATION_SECRET_MASK,
  destStatusVariant,
  parseLocalConfig,
  parseS3Config,
  parseWebDAVConfig,
  mutationSucceeded,
  resolveS3SecretUpdate,
  resolveSecretUpdate,
} from "./backup-destinations"

function makeDestination(overrides: Partial<BackupDestination> = {}): BackupDestination {
  return {
    id: 1,
    revision: 1,
    type: "local",
    enabled: true,
    config: JSON.stringify({ dir: "/backups", retention_count: 7 }),
    last_run_at: "",
    last_status: "",
    last_error: "",
    sort_order: 0,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    configured_secret_fields: [],
    ...overrides,
  }
}

describe("BackupDestination type shape", () => {
  it("has all required fields", () => {
    const dest = makeDestination()
    expect(dest.id).toBe(1)
    expect(dest.revision).toBe(1)
    expect(dest.type).toBe("local")
    expect(dest.enabled).toBe(true)
    expect(dest.configured_secret_fields).toEqual([])
  })
})

describe("parseLocalConfig", () => {
  it("parses a valid local config", () => {
    const dest = makeDestination({ config: JSON.stringify({ dir: "/data/backups", retention_count: 5 }) })
    const parsed = parseLocalConfig(dest)
    expect(parsed.dir).toBe("/data/backups")
    expect(parsed.retention_count).toBe(5)
  })

  it("returns empty object for invalid JSON", () => {
    const dest = makeDestination({ config: "not-json" })
    expect(parseLocalConfig(dest)).toEqual({})
  })

  it("returns empty object for non-local type", () => {
    const dest = makeDestination({ type: "s3", config: JSON.stringify({ dir: "/x" }) })
    expect(parseLocalConfig(dest)).toEqual({})
  })

  it("handles config with empty dir (server default)", () => {
    const dest = makeDestination({ config: JSON.stringify({ dir: "", retention_count: 7 }) })
    const parsed = parseLocalConfig(dest)
    expect(parsed.dir).toBe("")
    expect(parsed.retention_count).toBe(7)
  })
})

describe("buildLocalConfig", () => {
  it("serialises dir and retention_count", () => {
    const raw = buildLocalConfig("/srv/backups", 10)
    const parsed = JSON.parse(raw) as { dir: string; retention_count: number }
    expect(parsed.dir).toBe("/srv/backups")
    expect(parsed.retention_count).toBe(10)
  })

  it("round-trips through parseLocalConfig", () => {
    const config = buildLocalConfig("/tmp", 3)
    const dest = makeDestination({ config })
    const result = parseLocalConfig(dest)
    expect(result.dir).toBe("/tmp")
    expect(result.retention_count).toBe(3)
  })
})

describe("destStatusVariant", () => {
  it("returns secondary for success", () => {
    expect(destStatusVariant("success")).toBe("secondary")
  })

  it("returns destructive for failed", () => {
    expect(destStatusVariant("failed")).toBe("destructive")
  })

  it("returns outline for unknown / empty status", () => {
    expect(destStatusVariant("")).toBe("outline")
    expect(destStatusVariant("running")).toBe("outline")
  })
})

describe("BackupDestination configured_secret_fields", () => {
  it("is empty for local destinations (no secrets)", () => {
    const dest = makeDestination({ type: "local", configured_secret_fields: [] })
    expect(dest.configured_secret_fields).toHaveLength(0)
  })

  it("lists field names for destinations that have secrets configured", () => {
    const dest = makeDestination({
      type: "s3",
      configured_secret_fields: ["secret_access_key"],
    })
    expect(dest.configured_secret_fields).toContain("secret_access_key")
  })
})

describe("parseS3Config", () => {
  it("parses all required s3 fields", () => {
    const config = buildS3Config({
      endpoint: "s3.amazonaws.com",
      region: "",
      bucket: "my-bucket",
      prefix: "",
      accessKeyId: "AKID",
      secretAccessKey: "secret",
      useSsl: true,
      usePathStyle: false,
      retentionCount: 5,
    })
    const dest = makeDestination({ type: "s3", config })
    const parsed = parseS3Config(dest)
    expect(parsed.endpoint).toBe("s3.amazonaws.com")
    expect(parsed.bucket).toBe("my-bucket")
    expect(parsed.access_key_id).toBe("AKID")
    expect(parsed.secret_access_key).toBe("secret")
    expect(parsed.use_ssl).toBe(true)
    expect(parsed.use_path_style).toBe(false)
    expect(parsed.retention_count).toBe(5)
  })

  it("includes optional region and prefix when set", () => {
    const config = buildS3Config({
      endpoint: "minio.local:9000",
      region: "us-east-1",
      bucket: "backups",
      prefix: "prod/",
      accessKeyId: "user",
      secretAccessKey: "pass",
      useSsl: false,
      usePathStyle: true,
      retentionCount: 3,
    })
    const dest = makeDestination({ type: "s3", config })
    const parsed = parseS3Config(dest)
    expect(parsed.region).toBe("us-east-1")
    expect(parsed.prefix).toBe("prod/")
  })

  it("returns empty object for non-s3 type", () => {
    const dest = makeDestination({ type: "local", config: JSON.stringify({ endpoint: "x" }) })
    expect(parseS3Config(dest)).toEqual({})
  })

  it("returns empty object for invalid JSON", () => {
    const dest = makeDestination({ type: "s3", config: "not-json" })
    expect(parseS3Config(dest)).toEqual({})
  })
})

describe("buildS3Config", () => {
  const baseFields = {
    endpoint: "s3.amazonaws.com",
    region: "",
    bucket: "bucket",
    prefix: "",
    accessKeyId: "ID",
    secretAccessKey: "secret",
    useSsl: true,
    usePathStyle: false,
    retentionCount: 7,
  }

  it("serialises all required fields", () => {
    const obj = JSON.parse(buildS3Config(baseFields)) as Record<string, unknown>
    expect(obj.endpoint).toBe("s3.amazonaws.com")
    expect(obj.bucket).toBe("bucket")
    expect(obj.access_key_id).toBe("ID")
    expect(obj.secret_access_key).toBe("secret")
    expect(obj.use_ssl).toBe(true)
    expect(obj.use_path_style).toBe(false)
    expect(obj.retention_count).toBe(7)
  })

  it("excludes region key when region is empty", () => {
    const obj = JSON.parse(buildS3Config({ ...baseFields, region: "" })) as Record<string, unknown>
    expect(Object.prototype.hasOwnProperty.call(obj, "region")).toBe(false)
  })

  it("excludes prefix key when prefix is empty", () => {
    const obj = JSON.parse(buildS3Config({ ...baseFields, prefix: "" })) as Record<string, unknown>
    expect(Object.prototype.hasOwnProperty.call(obj, "prefix")).toBe(false)
  })

  it("includes region and prefix when provided", () => {
    const obj = JSON.parse(
      buildS3Config({ ...baseFields, region: "eu-west-1", prefix: "backups/" })
    ) as Record<string, unknown>
    expect(obj.region).toBe("eu-west-1")
    expect(obj.prefix).toBe("backups/")
  })
})

describe("resolveS3SecretUpdate (replacement-only)", () => {
  it("preserves the stored secret when mask is still shown", () => {
    const result = resolveS3SecretUpdate(DESTINATION_SECRET_MASK, false, true)
    expect(result.secret_access_key).toBe("")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("preserves the stored secret when the replacement field is left empty", () => {
    const result = resolveS3SecretUpdate("", true, true)
    expect(result.secret_access_key).toBe("")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("sends the new value when user typed a replacement secret", () => {
    const result = resolveS3SecretUpdate("newSecret123", true, true)
    expect(result.secret_access_key).toBe("newSecret123")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("sends the new value on create (not configured)", () => {
    const result = resolveS3SecretUpdate("mySecret", false, false)
    expect(result.secret_access_key).toBe("mySecret")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("does not emit cleared_secret_fields when not previously configured", () => {
    const result = resolveS3SecretUpdate("", false, false)
    expect(result.cleared_secret_fields).toBeUndefined()
  })
})

describe("mutationSucceeded", () => {
  it("only treats a returned mutation value as success", () => {
    expect(mutationSucceeded({ id: 1 })).toBe(true)
    expect(mutationSucceeded(undefined)).toBe(false)
    expect(mutationSucceeded(null as unknown as { id: number })).toBe(false)
  })
})

describe("resolveSecretUpdate (generic field-name-parameterized helper)", () => {
  it("preserves the stored secret when mask is still shown", () => {
    const result = resolveSecretUpdate("password", DESTINATION_SECRET_MASK, false, true)
    expect(result.value).toBe("")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("preserves the configured secret when an empty value is not being edited", () => {
    const result = resolveSecretUpdate("password", "", false, true)
    expect(result.value).toBe("")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("signals cleared_secret_fields when secret was explicitly cleared", () => {
    const result = resolveSecretUpdate("password", "", true, true)
    expect(result.value).toBe("")
    expect(result.cleared_secret_fields).toEqual(["password"])
  })

  it("sends the new value when user typed a replacement secret", () => {
    const result = resolveSecretUpdate("password", "newPass123", true, true)
    expect(result.value).toBe("newPass123")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("sends the new value on create (not configured)", () => {
    const result = resolveSecretUpdate("password", "myPass", false, false)
    expect(result.value).toBe("myPass")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("does not emit cleared_secret_fields when not previously configured", () => {
    const result = resolveSecretUpdate("password", "", false, false)
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("uses the supplied fieldName in cleared_secret_fields", () => {
    const result = resolveSecretUpdate("my_custom_field", "", true, true)
    expect(result.cleared_secret_fields).toEqual(["my_custom_field"])
  })
})

describe("parseWebDAVConfig", () => {
  it("parses all webdav fields", () => {
    const config = buildWebDAVConfig({
      url: "https://dav.example.com/remote.php/dav/files/user",
      path: "backups",
      username: "admin",
      password: "secret",
      retentionCount: 5,
    })
    const dest = makeDestination({ type: "webdav", config })
    const parsed = parseWebDAVConfig(dest)
    expect(parsed.url).toBe("https://dav.example.com/remote.php/dav/files/user")
    expect(parsed.path).toBe("backups")
    expect(parsed.username).toBe("admin")
    expect(parsed.password).toBe("secret")
    expect(parsed.retention_count).toBe(5)
  })

  it("returns empty object for non-webdav type", () => {
    const dest = makeDestination({ type: "local", config: JSON.stringify({ url: "https://x" }) })
    expect(parseWebDAVConfig(dest)).toEqual({})
  })

  it("returns empty object for invalid JSON", () => {
    const dest = makeDestination({ type: "webdav", config: "not-json" })
    expect(parseWebDAVConfig(dest)).toEqual({})
  })
})

describe("buildWebDAVConfig", () => {
  const baseFields = {
    url: "https://dav.example.com/remote.php/dav/files/user",
    path: "",
    username: "",
    password: "secret",
    retentionCount: 7,
  }

  it("serialises required fields", () => {
    const obj = JSON.parse(buildWebDAVConfig(baseFields)) as Record<string, unknown>
    expect(obj.url).toBe("https://dav.example.com/remote.php/dav/files/user")
    expect(obj.password).toBe("secret")
    expect(obj.retention_count).toBe(7)
  })

  it("excludes path key when path is empty", () => {
    const obj = JSON.parse(buildWebDAVConfig({ ...baseFields, path: "" })) as Record<string, unknown>
    expect(Object.prototype.hasOwnProperty.call(obj, "path")).toBe(false)
  })

  it("excludes username key when username is empty", () => {
    const obj = JSON.parse(buildWebDAVConfig({ ...baseFields, username: "" })) as Record<string, unknown>
    expect(Object.prototype.hasOwnProperty.call(obj, "username")).toBe(false)
  })

  it("includes path and username when provided", () => {
    const obj = JSON.parse(
      buildWebDAVConfig({ ...baseFields, path: "backups/prod", username: "admin" })
    ) as Record<string, unknown>
    expect(obj.path).toBe("backups/prod")
    expect(obj.username).toBe("admin")
  })

  it("round-trips through parseWebDAVConfig", () => {
    const fields = {
      url: "https://dav.example.com/dav",
      path: "mypath",
      username: "user",
      password: "pass",
      retentionCount: 3,
    }
    const config = buildWebDAVConfig(fields)
    const dest = makeDestination({ type: "webdav", config })
    const parsed = parseWebDAVConfig(dest)
    expect(parsed.url).toBe(fields.url)
    expect(parsed.path).toBe(fields.path)
    expect(parsed.username).toBe(fields.username)
    expect(parsed.password).toBe(fields.password)
    expect(parsed.retention_count).toBe(fields.retentionCount)
  })
})

describe("webdav password secret resolution (via resolveSecretUpdate)", () => {
  it("preserves the stored password when mask is still shown", () => {
    const result = resolveSecretUpdate("password", DESTINATION_SECRET_MASK, false, true)
    expect(result.value).toBe("")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("signals cleared_secret_fields=['password'] when password was explicitly cleared", () => {
    const result = resolveSecretUpdate("password", "", true, true)
    expect(result.value).toBe("")
    expect(result.cleared_secret_fields).toEqual(["password"])
  })

  it("sends the new password when user typed a replacement", () => {
    const result = resolveSecretUpdate("password", "newPass", true, true)
    expect(result.value).toBe("newPass")
    expect(result.cleared_secret_fields).toBeUndefined()
  })

  it("sends the password on create (not configured)", () => {
    const result = resolveSecretUpdate("password", "myPass", false, false)
    expect(result.value).toBe("myPass")
    expect(result.cleared_secret_fields).toBeUndefined()
  })
})
