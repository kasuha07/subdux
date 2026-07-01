import { describe, expect, it } from "vitest"

import {
  computePasswordVerificationValue,
  detectZipEncryption,
  verifyZipPassword,
  type EncryptedZipEntry,
} from "@/lib/zip-encryption"

// A small File polyfill backed by an ArrayBuffer, sufficient for slice-based
// reads used by the zip-encryption utilities in the node test environment.
function makeFile(bytes: Uint8Array): File {
  const buffer = new ArrayBuffer(bytes.byteLength)
  new Uint8Array(buffer).set(bytes)
  return new File([buffer], "backup.zip", { type: "application/zip" })
}

// PBKDF2-HMAC-SHA1(password, salt, 1000, keySize*2+2) with keySize=32 (AES-256).
// The last 2 bytes of the derived key are the password verification value.
describe("computePasswordVerificationValue", () => {
  it("derives a stable 2-byte pwv for a known password/salt/strength", async () => {
    const salt = new Uint8Array([
      0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e,
      0x0f,
    ])
    const pwv = await computePasswordVerificationValue("test", salt, 3)
    expect(pwv).toBeInstanceOf(Uint8Array)
    expect(pwv.length).toBe(2)

    // Deterministic: re-deriving yields the identical value.
    const pwv2 = await computePasswordVerificationValue("test", salt, 3)
    expect(Array.from(pwv2)).toEqual(Array.from(pwv))

    // A different password yields a (near-certainly) different pwv.
    const other = await computePasswordVerificationValue("wrong", salt, 3)
    expect(Array.from(other)).not.toEqual(Array.from(pwv))
  })

  it("uses the correct salt length per AES strength", async () => {
    // AES-128 uses an 8-byte salt; AES-256 uses a 16-byte salt. The function
    // consumes whatever salt is passed, so verify both strengths produce a pwv.
    const salt128 = new Uint8Array(8).fill(0xaa)
    const pwv128 = await computePasswordVerificationValue("pw", salt128, 1)
    expect(pwv128.length).toBe(2)

    const salt256 = new Uint8Array(16).fill(0xbb)
    const pwv256 = await computePasswordVerificationValue("pw", salt256, 3)
    expect(pwv256.length).toBe(2)
  })
})

// Build a minimal but structurally valid WinZip-AES encrypted ZIP:
//   [local header][name][AES extra][salt][pwv][ciphertext][authcode]
//   [central header][name][AES extra]
//   [EOCD]
async function buildEncryptedZipFixture(options: {
  password: string
  aesStrength: number
}): Promise<{ bytes: Uint8Array; localHeaderOffset: number }> {
  const { password, aesStrength } = options
  const keySize = aesStrength === 1 ? 16 : aesStrength === 2 ? 24 : 32
  const saltLen = keySize / 2

  const name = new TextEncoder().encode("db")
  const salt = new Uint8Array(saltLen)
  for (let i = 0; i < saltLen; i += 1) {
    salt[i] = (i * 7 + 3) & 0xff
  }
  const pwv = await computePasswordVerificationValue(password, salt, aesStrength)
  const ciphertext = new Uint8Array([0x11, 0x22, 0x33, 0x44])
  const authcode = new Uint8Array(10).fill(0x99)

  // AES extra field: id(2) size(2)=7, version(2), "AE"(2), strength(1), method(2)
  const aesExtra = new Uint8Array(11)
  const aeDv = new DataView(aesExtra.buffer)
  aeDv.setUint16(0, 0x9901, true)
  aeDv.setUint16(2, 7, true)
  aeDv.setUint16(4, 2, true) // version AE-2
  aesExtra[6] = 0x41 // 'A'
  aesExtra[7] = 0x45 // 'E'
  aesExtra[8] = aesStrength
  aeDv.setUint16(9, 0, true) // actual method = stored

  const entryData = new Uint8Array(salt.length + 2 + ciphertext.length + authcode.length)
  entryData.set(salt, 0)
  entryData.set(pwv, salt.length)
  entryData.set(ciphertext, salt.length + 2)
  entryData.set(authcode, salt.length + 2 + ciphertext.length)

  // Local file header (30 bytes fixed).
  const local = new Uint8Array(30)
  const lDv = new DataView(local.buffer)
  lDv.setUint32(0, 0x04034b50, true)
  lDv.setUint16(4, 51, true) // version needed
  lDv.setUint16(6, 0x0001, true) // gp flag bit0 = encrypted
  lDv.setUint16(8, 99, true) // compression method = AES
  lDv.setUint16(26, name.length, true)
  lDv.setUint16(28, aesExtra.length, true)

  const localHeaderOffset = 0
  const localBlock = concat([local, name, aesExtra, entryData])

  // Central directory header (46 bytes fixed).
  const central = new Uint8Array(46)
  const cDv = new DataView(central.buffer)
  cDv.setUint32(0, 0x02014b50, true)
  cDv.setUint16(8, 0x0001, true) // gp flag bit0 = encrypted
  cDv.setUint16(10, 99, true) // compression method = AES
  cDv.setUint16(28, name.length, true)
  cDv.setUint16(30, aesExtra.length, true)
  cDv.setUint16(32, 0, true) // comment length
  cDv.setUint32(42, localHeaderOffset, true)

  const centralBlock = concat([central, name, aesExtra])
  const cdOffset = localBlock.length
  const cdSize = centralBlock.length

  // EOCD (22 bytes fixed, no comment).
  const eocd = new Uint8Array(22)
  const eDv = new DataView(eocd.buffer)
  eDv.setUint32(0, 0x06054b50, true)
  eDv.setUint16(8, 1, true) // entries on this disk
  eDv.setUint16(10, 1, true) // total entries
  eDv.setUint32(12, cdSize, true)
  eDv.setUint32(16, cdOffset, true)

  return { bytes: concat([localBlock, centralBlock, eocd]), localHeaderOffset }
}

function concat(parts: Uint8Array[]): Uint8Array {
  const total = parts.reduce((sum, p) => sum + p.length, 0)
  const out = new Uint8Array(total)
  let offset = 0
  for (const p of parts) {
    out.set(p, offset)
    offset += p.length
  }
  return out
}

describe("detectZipEncryption", () => {
  it("detects a WinZip-AES encrypted entry via EOCD + central directory parsing", async () => {
    const { bytes, localHeaderOffset } = await buildEncryptedZipFixture({
      password: "hunter2",
      aesStrength: 3,
    })
    const result = await detectZipEncryption(makeFile(bytes))
    expect(result.isZip).toBe(true)
    expect(result.encrypted).toBe(true)
    expect(result.firstEncryptedEntry).toBeDefined()
    expect(result.firstEncryptedEntry?.localHeaderOffset).toBe(localHeaderOffset)
    expect(result.firstEncryptedEntry?.aesStrength).toBe(3)
  })

  it("reports non-zip for a tiny non-zip file", async () => {
    const result = await detectZipEncryption(makeFile(new Uint8Array([1, 2, 3])))
    expect(result.isZip).toBe(false)
    expect(result.encrypted).toBe(false)
  })

  it("reports an unencrypted zip when no entry has the encryption flag", async () => {
    const name = new TextEncoder().encode("db")
    const local = new Uint8Array(30)
    new DataView(local.buffer).setUint32(0, 0x04034b50, true)
    new DataView(local.buffer).setUint16(26, name.length, true)
    const localBlock = concat([local, name, new Uint8Array([0xde, 0xad])])

    const central = new Uint8Array(46)
    const cDv = new DataView(central.buffer)
    cDv.setUint32(0, 0x02014b50, true)
    cDv.setUint16(28, name.length, true)
    const centralBlock = concat([central, name])

    const eocd = new Uint8Array(22)
    const eDv = new DataView(eocd.buffer)
    eDv.setUint32(0, 0x06054b50, true)
    eDv.setUint16(8, 1, true)
    eDv.setUint16(10, 1, true)
    eDv.setUint32(12, centralBlock.length, true)
    eDv.setUint32(16, localBlock.length, true)

    const result = await detectZipEncryption(makeFile(concat([localBlock, centralBlock, eocd])))
    expect(result.isZip).toBe(true)
    expect(result.encrypted).toBe(false)
    expect(result.firstEncryptedEntry).toBeUndefined()
  })
})

describe("verifyZipPassword", () => {
  it("returns valid=true for the correct password", async () => {
    const { bytes } = await buildEncryptedZipFixture({ password: "correct-horse", aesStrength: 3 })
    const detected = await detectZipEncryption(makeFile(bytes))
    const entry = detected.firstEncryptedEntry as EncryptedZipEntry
    const result = await verifyZipPassword(makeFile(bytes), entry, "correct-horse")
    expect(result.verified).toBe(true)
    expect(result.valid).toBe(true)
  })

  it("returns valid=false for the wrong password", async () => {
    const { bytes } = await buildEncryptedZipFixture({ password: "correct-horse", aesStrength: 3 })
    const detected = await detectZipEncryption(makeFile(bytes))
    const entry = detected.firstEncryptedEntry as EncryptedZipEntry
    const result = await verifyZipPassword(makeFile(bytes), entry, "wrong-password")
    expect(result.verified).toBe(true)
    expect(result.valid).toBe(false)
  })

  it("works for AES-128 strength as well", async () => {
    const { bytes } = await buildEncryptedZipFixture({ password: "pw128", aesStrength: 1 })
    const detected = await detectZipEncryption(makeFile(bytes))
    expect(detected.firstEncryptedEntry?.aesStrength).toBe(1)
    const entry = detected.firstEncryptedEntry as EncryptedZipEntry
    const ok = await verifyZipPassword(makeFile(bytes), entry, "pw128")
    expect(ok.valid).toBe(true)
    const bad = await verifyZipPassword(makeFile(bytes), entry, "nope")
    expect(bad.valid).toBe(false)
  })

  it("does not hard-block on a structurally invalid local header", async () => {
    const bytes = new Uint8Array(64).fill(0)
    const entry: EncryptedZipEntry = { localHeaderOffset: 0, aesStrength: 3 }
    const result = await verifyZipPassword(makeFile(bytes), entry, "anything")
    expect(result.verified).toBe(false)
  })
})
