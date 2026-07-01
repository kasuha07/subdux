// Client-side ZIP encryption detection and WinZip-AES password verification.
//
// The goal is to decide whether an uploaded backup ZIP is encrypted, and if so,
// verify the user's password WITHOUT decompressing or loading the whole file
// into memory. Only tiny `File.slice(...)` reads are performed:
//   - the trailing End Of Central Directory record + central directory
//   - the local file header + salt/pwv prefix of the first encrypted entry
//
// The WinZip-AES verification maps byte-exactly to the backend implementation in
// github.com/yeka/zip (crypto.go): PBKDF2-HMAC-SHA1(password, salt, 1000,
// keySize*2 + 2), where the last 2 derived bytes are the password verification
// value (pwv) and the salt length is keySize/2.

const EOCD_SIGNATURE = 0x06054b50
const CENTRAL_HEADER_SIGNATURE = 0x02014b50
const LOCAL_HEADER_SIGNATURE = 0x04034b50
const AES_EXTRA_FIELD_ID = 0x9901
const AES_COMPRESSION_METHOD = 99
const ZIP64_SENTINEL_32 = 0xffffffff

// EOCD is 22 bytes fixed + up to 65535 bytes of comment.
const MAX_EOCD_SEARCH = 22 + 65535

export interface EncryptedZipEntry {
  localHeaderOffset: number
  // AES strength: 1 = 128-bit, 2 = 192-bit, 3 = 256-bit.
  aesStrength: number
}

export interface DetectZipEncryptionResult {
  isZip: boolean
  encrypted: boolean
  // Present only when encryption was detected and the entry could be located.
  firstEncryptedEntry?: EncryptedZipEntry
  // True when the tail could not be parsed (not a ZIP, ZIP64, truncated, etc.).
  // In this case `encrypted` is best-effort and the caller should fall back to
  // server-side validation rather than hard-blocking.
  cannotVerify?: boolean
}

// keySize (in bytes) of the AES key for each WinZip-AES strength.
function keySizeForStrength(strength: number): number | null {
  if (strength === 1) {
    return 16
  }
  if (strength === 2) {
    return 24
  }
  if (strength === 3) {
    return 32
  }
  return null
}

async function readSlice(file: File, start: number, end: number): Promise<DataView> {
  const buffer = await file.slice(start, end).arrayBuffer()
  return new DataView(buffer)
}

// Locate the EOCD record inside a tail buffer and return its start offset,
// or -1 if not found. Scans backwards from the end.
function findEocdOffset(view: DataView): number {
  // The EOCD signature is 4 bytes; the earliest position it can start is such
  // that the 22-byte fixed record still fits.
  for (let offset = view.byteLength - 22; offset >= 0; offset -= 1) {
    if (view.getUint32(offset, true) === EOCD_SIGNATURE) {
      return offset
    }
  }
  return -1
}

export async function detectZipEncryption(file: File): Promise<DetectZipEncryptionResult> {
  if (file.size < 22) {
    return { isZip: false, encrypted: false }
  }

  try {
    const tailStart = Math.max(0, file.size - MAX_EOCD_SEARCH)
    const tailView = await readSlice(file, tailStart, file.size)

    const eocdOffset = findEocdOffset(tailView)
    if (eocdOffset < 0) {
      return { isZip: false, encrypted: false }
    }

    // EOCD fields (relative to the EOCD start):
    //   offset 10: total number of central directory entries (uint16)
    //   offset 12: size of central directory (uint32)
    //   offset 16: offset of central directory from start of archive (uint32)
    const totalEntries = tailView.getUint16(eocdOffset + 10, true)
    const cdSize = tailView.getUint32(eocdOffset + 12, true)
    const cdOffset = tailView.getUint32(eocdOffset + 16, true)

    // ZIP64: the 32-bit fields are sentinels. We do not parse ZIP64 for these
    // backups; treat as "cannot verify" and let the server validate.
    if (
      cdSize === ZIP64_SENTINEL_32 ||
      cdOffset === ZIP64_SENTINEL_32 ||
      totalEntries === 0xffff
    ) {
      return { isZip: true, encrypted: false, cannotVerify: true }
    }

    if (cdSize === 0 || cdOffset + cdSize > file.size) {
      return { isZip: true, encrypted: false, cannotVerify: true }
    }

    const cdView = await readSlice(file, cdOffset, cdOffset + cdSize)

    let pos = 0
    while (pos + 46 <= cdView.byteLength) {
      if (cdView.getUint32(pos, true) !== CENTRAL_HEADER_SIGNATURE) {
        break
      }

      // Central directory file header fields (relative to header start):
      //   offset 8:  general purpose bit flag (uint16), bit 0 = encrypted
      //   offset 10: compression method (uint16), 99 = WinZip AES
      //   offset 28: file name length (uint16)
      //   offset 30: extra field length (uint16)
      //   offset 32: file comment length (uint16)
      //   offset 42: relative offset of local header (uint32)
      const gpFlag = cdView.getUint16(pos + 8, true)
      const compressionMethod = cdView.getUint16(pos + 10, true)
      const nameLen = cdView.getUint16(pos + 28, true)
      const extraLen = cdView.getUint16(pos + 30, true)
      const commentLen = cdView.getUint16(pos + 32, true)
      const localHeaderOffset = cdView.getUint32(pos + 42, true)

      const isEncrypted = (gpFlag & 0x1) === 0x1

      if (isEncrypted && compressionMethod === AES_COMPRESSION_METHOD) {
        // Parse the AES extra field (header id 0x9901) to get the strength.
        const extraStart = pos + 46 + nameLen
        const aesStrength = parseAesStrengthFromExtra(cdView, extraStart, extraLen)
        if (aesStrength !== null) {
          return {
            isZip: true,
            encrypted: true,
            firstEncryptedEntry: { localHeaderOffset, aesStrength },
          }
        }
      }

      pos += 46 + nameLen + extraLen + commentLen
    }

    return { isZip: true, encrypted: false }
  } catch {
    return { isZip: false, encrypted: false, cannotVerify: true }
  }
}

// Walk an extra field region looking for the WinZip-AES header (0x9901) and
// return its strength byte (1/2/3), or null if not present/malformed.
function parseAesStrengthFromExtra(
  view: DataView,
  start: number,
  length: number
): number | null {
  let pos = start
  const end = start + length
  while (pos + 4 <= end && pos + 4 <= view.byteLength) {
    const headerId = view.getUint16(pos, true)
    const dataSize = view.getUint16(pos + 2, true)
    if (headerId === AES_EXTRA_FIELD_ID) {
      // AES extra field data layout:
      //   offset 0: version (uint16)
      //   offset 2: vendor id "AE" (uint16)
      //   offset 4: AES strength (uint8) -- 1/2/3
      //   offset 5: actual compression method (uint16)
      if (pos + 4 + 5 <= view.byteLength) {
        return view.getUint8(pos + 4 + 4)
      }
      return null
    }
    pos += 4 + dataSize
  }
  return null
}

// Thrown when Web Crypto (crypto.subtle) is unavailable so the caller can fall
// back to server-side validation instead of blocking the user.
export class WebCryptoUnavailableError extends Error {
  constructor() {
    super("Web Crypto is unavailable")
    this.name = "WebCryptoUnavailableError"
  }
}

// Compute the WinZip-AES password verification value (last 2 bytes of the
// derived key) for the given password/salt/strength. Exposed for testing.
export async function computePasswordVerificationValue(
  password: string,
  salt: Uint8Array,
  aesStrength: number
): Promise<Uint8Array> {
  const keySize = keySizeForStrength(aesStrength)
  if (keySize === null) {
    throw new Error("unsupported AES strength")
  }

  const subtle = globalThis.crypto?.subtle
  if (!subtle) {
    throw new WebCryptoUnavailableError()
  }

  const pwBytes = new TextEncoder().encode(password)
  const baseKey = await subtle.importKey("raw", pwBytes, "PBKDF2", false, ["deriveBits"])
  // Derived length = keySize*2 (enc key + auth key) + 2 (pwv), in bits.
  const dkLenBits = (keySize * 2 + 2) * 8
  const derived = await subtle.deriveBits(
    {
      name: "PBKDF2",
      hash: "SHA-1",
      // A fresh copy avoids ArrayBuffer/SharedArrayBuffer typing friction.
      salt: salt.slice(),
      iterations: 1000,
    },
    baseKey,
    dkLenBits
  )

  const derivedBytes = new Uint8Array(derived)
  return derivedBytes.slice(derivedBytes.length - 2)
}

function constantTimeEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false
  }
  let diff = 0
  for (let i = 0; i < a.length; i += 1) {
    diff |= a[i] ^ b[i]
  }
  return diff === 0
}

export interface VerifyZipPasswordResult {
  // Whether client-side verification could actually run. When false, the caller
  // should let the server perform the real validation.
  verified: boolean
  // Only meaningful when `verified` is true.
  valid: boolean
}

// Verify a WinZip-AES password against the first encrypted entry without
// decrypting any data. Reads only the local header + salt/pwv prefix.
export async function verifyZipPassword(
  file: File,
  entry: EncryptedZipEntry,
  password: string
): Promise<VerifyZipPasswordResult> {
  const keySize = keySizeForStrength(entry.aesStrength)
  if (keySize === null) {
    return { verified: false, valid: false }
  }

  try {
    const headerView = await readSlice(file, entry.localHeaderOffset, entry.localHeaderOffset + 30)
    if (headerView.byteLength < 30) {
      return { verified: false, valid: false }
    }
    if (headerView.getUint32(0, true) !== LOCAL_HEADER_SIGNATURE) {
      return { verified: false, valid: false }
    }

    // Local file header fields:
    //   offset 26: file name length (uint16)
    //   offset 28: extra field length (uint16)
    const localNameLen = headerView.getUint16(26, true)
    const localExtraLen = headerView.getUint16(28, true)
    const dataStart = entry.localHeaderOffset + 30 + localNameLen + localExtraLen

    // WinZip-AES data layout: [salt][pwv (2 bytes)][ciphertext][authcode (10)].
    const saltLen = keySize / 2
    const prefixView = await readSlice(file, dataStart, dataStart + saltLen + 2)
    if (prefixView.byteLength < saltLen + 2) {
      return { verified: false, valid: false }
    }

    const salt = new Uint8Array(prefixView.buffer, prefixView.byteOffset, saltLen)
    const storedPwv = new Uint8Array(prefixView.buffer, prefixView.byteOffset + saltLen, 2)

    const computedPwv = await computePasswordVerificationValue(password, salt, entry.aesStrength)
    return { verified: true, valid: constantTimeEqual(computedPwv, storedPwv) }
  } catch (error) {
    if (error instanceof WebCryptoUnavailableError) {
      return { verified: false, valid: false }
    }
    // Structural error: do not hard-block; let the server validate.
    return { verified: false, valid: false }
  }
}
