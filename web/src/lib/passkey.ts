type PublicKeyCredentialDescriptorJSON = {
  type: PublicKeyCredentialType
  id: string
  transports?: AuthenticatorTransport[]
}

type PublicKeyCredentialUserEntityJSON = {
  id: string
  name: string
  displayName: string
}

type PublicKeyCredentialCreationOptionsJSON = {
  rp: PublicKeyCredentialRpEntity
  user: PublicKeyCredentialUserEntityJSON
  challenge: string
  pubKeyCredParams: PublicKeyCredentialParameters[]
  timeout?: number
  excludeCredentials?: PublicKeyCredentialDescriptorJSON[]
  authenticatorSelection?: AuthenticatorSelectionCriteria
  attestation?: AttestationConveyancePreference
  extensions?: AuthenticationExtensionsClientInputs
}

type PublicKeyCredentialRequestOptionsJSON = {
  challenge: string
  timeout?: number
  rpId?: string
  allowCredentials?: PublicKeyCredentialDescriptorJSON[]
  userVerification?: UserVerificationRequirement
  extensions?: AuthenticationExtensionsClientInputs
}

export type CredentialCreationJSON = {
  publicKey: PublicKeyCredentialCreationOptionsJSON
  mediation?: CredentialMediationRequirement
}

export type CredentialAssertionJSON = {
  publicKey: PublicKeyCredentialRequestOptionsJSON
  mediation?: CredentialMediationRequirement
}

type PasskeyCredentialBase = {
  id: string
  rawId: string
  type: string
  authenticatorAttachment?: string | null
  clientExtensionResults: AuthenticationExtensionsClientOutputs
}

export type PasskeyCreateCredential = PasskeyCredentialBase & {
  response: {
    attestationObject: string
    clientDataJSON: string
    transports?: string[]
  }
}

export type PasskeyGetCredential = PasskeyCredentialBase & {
  response: {
    authenticatorData: string
    clientDataJSON: string
    signature: string
    userHandle?: string | null
  }
}

export function isPasskeySupported(): boolean {
  return typeof window !== "undefined" && window.isSecureContext && "PublicKeyCredential" in window
}

export async function createPasskeyCredential(options: CredentialCreationJSON): Promise<PasskeyCreateCredential> {
  if (!isPasskeySupported()) {
    throw new Error("Passkey is not supported on this device or browser")
  }

  const credential = await navigator.credentials.create({
    publicKey: {
      ...options.publicKey,
      challenge: decodeBase64URL(options.publicKey.challenge),
      user: {
        ...options.publicKey.user,
        id: decodeBase64URL(options.publicKey.user.id),
      },
      excludeCredentials: options.publicKey.excludeCredentials?.map((item) => ({
        ...item,
        id: decodeBase64URL(item.id),
      })),
    },
  })

  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error("Failed to create passkey")
  }
  if (!(credential.response instanceof AuthenticatorAttestationResponse)) {
    throw new Error("Unexpected passkey attestation response")
  }

  return {
    id: credential.id,
    rawId: encodeBase64URL(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      attestationObject: encodeBase64URL(credential.response.attestationObject),
      clientDataJSON: encodeBase64URL(credential.response.clientDataJSON),
      transports: typeof credential.response.getTransports === "function" ? credential.response.getTransports() : undefined,
    },
  }
}

export async function getPasskeyCredential(options: CredentialAssertionJSON): Promise<PasskeyGetCredential> {
  if (!isPasskeySupported()) {
    throw new Error("Passkey is not supported on this device or browser")
  }

  const credential = await navigator.credentials.get({
    mediation: options.mediation,
    publicKey: {
      ...options.publicKey,
      challenge: decodeBase64URL(options.publicKey.challenge),
      allowCredentials: options.publicKey.allowCredentials?.map((item) => ({
        ...item,
        id: decodeBase64URL(item.id),
      })),
    },
  })

  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error("Failed to verify passkey")
  }
  if (!(credential.response instanceof AuthenticatorAssertionResponse)) {
    throw new Error("Unexpected passkey assertion response")
  }

  return {
    id: credential.id,
    rawId: encodeBase64URL(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      authenticatorData: encodeBase64URL(credential.response.authenticatorData),
      clientDataJSON: encodeBase64URL(credential.response.clientDataJSON),
      signature: encodeBase64URL(credential.response.signature),
      userHandle: credential.response.userHandle ? encodeBase64URL(credential.response.userHandle) : null,
    },
  }
}

function decodeBase64URL(input: string): ArrayBuffer {
  const normalized = normalizeBase64(input)
  const raw = atob(normalized)
  const bytes = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i += 1) {
    bytes[i] = raw.charCodeAt(i)
  }
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

function encodeBase64URL(value: ArrayBuffer | ArrayBufferView): string {
  const bytes = value instanceof ArrayBuffer
    ? new Uint8Array(value)
    : new Uint8Array(value.buffer, value.byteOffset, value.byteLength)
  let binary = ""
  for (const byte of bytes) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/u, "")
}

function normalizeBase64(input: string): string {
  const converted = input.replace(/-/g, "+").replace(/_/g, "/")
  const padding = converted.length % 4
  if (padding === 0) {
    return converted
  }
  return `${converted}${"=".repeat(4 - padding)}`
}
